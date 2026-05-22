// seed 是一次性数据初始化工具，向 Neo4j 和 PostgreSQL 写入测试数据。
// MERGE / ON CONFLICT 保证幂等，重复执行安全。
//
// 用法：
//
//	go run ./cmd/seed/...
//	go run ./cmd/seed/... -config configs/config.yaml
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"

	neo4jdb "matrix/api/internal/infra/neo4j"
	"matrix/api/internal/infra/postgres"
	"matrix/api/internal/repository/pg_repo"
	"matrix/api/pkg/auth"
	"matrix/api/pkg/config"
	"matrix/api/pkg/log"
)

// ── 设备种子数据 ────────────────────────────────────────────────────────────

type deviceSeed struct {
	id       string
	name     string
	typ      string
	typeCN   string
	vendor   string
	vendorCN string
	mip      string
}

var deviceSeeds = []deviceSeed{
	{"dev-router-wan-01", "router-wan-01", "router", "广域网路由器", "Cisco", "思科", "10.0.0.1"},
	{"dev-fw-01", "fw-01", "firewall", "防火墙", "H3C", "新华三", "10.0.1.1"},
	{"dev-core-sw-01", "core-sw-01", "switch", "核心交换机", "H3C", "新华三", "10.1.0.1"},
	{"dev-core-sw-02", "core-sw-02", "switch", "核心交换机", "H3C", "新华三", "10.1.0.2"},
	{"dev-acc-sw-f1", "acc-sw-floor1", "switch", "接入层交换机", "H3C", "新华三", "10.2.1.1"},
}

// connections: [fromID, toID] 描述 CONNECTED_TO 边
var connSeeds = [][2]string{
	{"dev-router-wan-01", "dev-fw-01"},
	{"dev-fw-01", "dev-core-sw-01"},
	{"dev-fw-01", "dev-core-sw-02"},
	{"dev-core-sw-01", "dev-acc-sw-f1"},
}

// ── 部门种子数据 ────────────────────────────────────────────────────────────

var deptSeeds = []pg_repo.Department{
	{ID: "00000000-0000-0000-0001-000000000001", ExternalID: "od-dept-001", Name: "研发中心", MemberCount: 20},
	{ID: "00000000-0000-0000-0001-000000000002", ExternalID: "od-dept-002", Name: "IT运维部", MemberCount: 8},
}

// ── 员工种子数据 ────────────────────────────────────────────────────────────

var empSeeds = []pg_repo.Employee{
	{
		ID: "00000000-0000-0000-0002-000000000001", ExternalID: "ou-emp-001",
		Name: "张工", Email: "zhangwork@company.com", Mobile: "13800000001",
		Status: 1, DepartmentIDs: []string{"od-dept-001"},
	},
	{
		ID: "00000000-0000-0000-0002-000000000002", ExternalID: "ou-emp-002",
		Name: "李工", Email: "liwork@company.com", Mobile: "13800000002",
		Status: 1, DepartmentIDs: []string{"od-dept-002"},
	},
	{
		ID: "00000000-0000-0000-0002-000000000003", ExternalID: "ou-emp-003",
		Name: "王工", Email: "wangwork@company.com", Mobile: "13800000003",
		Status: 1, DepartmentIDs: []string{"od-dept-001"},
	},
	{
		ID: "00000000-0000-0000-0002-000000000004", ExternalID: "ou-emp-004",
		Name: "赵工", Email: "zhaowork@company.com", Mobile: "13800000004",
		Status: 1, DepartmentIDs: []string{"od-dept-002"},
	},
}

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	log.Config = &cfg.Log
	log.Config.Reset()

	ctx := context.Background()

	// ── PostgreSQL ──────────────────────────────────────────────────────────
	db, err := postgres.Open(cfg.Postgres.DSN())
	if err != nil {
		log.Logger.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	if err := postgres.RunMigrationsEmbedded(cfg.Postgres.DSN()); err != nil {
		log.Logger.Fatalf("run migrations: %v", err)
	}
	log.Logger.Info("postgres migrations applied")

	if err := seedPostgres(ctx, db); err != nil {
		log.Logger.Fatalf("seed postgres: %v", err)
	}
	log.Logger.Infof("postgres: seeded %d departments, %d employees, admin user", len(deptSeeds), len(empSeeds))

	// ── Neo4j ────────────────────────────────────────────────────────────────
	driver, err := neo4jdb.Open(cfg.Neo4j)
	if err != nil {
		log.Logger.Fatalf("connect neo4j: %v", err)
	}
	if driver == nil {
		log.Logger.Warn("neo4j URI 未配置，跳过 Neo4j seed")
		return
	}
	defer driver.Close(ctx)

	if err := neo4jdb.InitSchema(ctx, driver, cfg.Neo4j.Database); err != nil {
		log.Logger.Fatalf("neo4j init schema: %v", err)
	}

	if err := seedNeo4j(ctx, driver, cfg.Neo4j.Database); err != nil {
		log.Logger.Fatalf("seed neo4j: %v", err)
	}
	log.Logger.Infof("neo4j: seeded %d devices, %d connections", len(deviceSeeds), len(connSeeds))

	log.Logger.Info("✓ seed 完成")
}

// seedPostgres 写入部门、员工和默认管理员账号，ON CONFLICT DO UPDATE 保证幂等。
func seedPostgres(ctx context.Context, db *sqlx.DB) error {
	deptRepo := pg_repo.NewDepartmentRepository(db)
	empRepo := pg_repo.NewEmployeeRepository(db)
	userRepo := pg_repo.NewAuthUserRepository(db)

	if err := deptRepo.SaveBatch(ctx, deptSeeds); err != nil {
		return fmt.Errorf("seed departments: %w", err)
	}
	if err := empRepo.SaveBatch(ctx, empSeeds); err != nil {
		return fmt.Errorf("seed employees: %w", err)
	}
	if err := seedAdminUser(ctx, userRepo); err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}
	return nil
}

// seedAdminUser 写入默认管理员，用户名已存在则跳过（不覆盖已修改的密码）。
func seedAdminUser(ctx context.Context, repo pg_repo.AuthUserRepository) error {
	existing, err := repo.FindByUsername(ctx, "admin")
	if err != nil {
		return err
	}
	if existing != nil {
		log.Logger.Info("admin user already exists, skip")
		return nil
	}
	hash, err := auth.HashPassword("admin123")
	if err != nil {
		return err
	}
	_, err = repo.Create(ctx, "admin", hash, []string{"admin"})
	if err != nil {
		return err
	}
	log.Logger.Info("admin user created (username=admin password=admin123)")
	return nil
}

// seedNeo4j 写入 Device 节点和 CONNECTED_TO 关系，MERGE 保证幂等。
func seedNeo4j(ctx context.Context, driver neo4j.Driver, dbName string) error {
	now := time.Now().UTC()

	// 写入设备节点
	for _, d := range deviceSeeds {
		_, err := neo4j.ExecuteQuery(ctx, driver,
			`MERGE (d:Device {id: $id})
			 ON CREATE SET
			   d.name = $name, d.type = $type, d.type_cn = $typeCN,
			   d.vendor = $vendor, d.vendor_cn = $vendorCN, d.mip = $mip,
			   d.version = 1, d.create_by = 'seed', d.update_by = 'seed',
			   d.created_at = $now, d.updated_at = $now
			 ON MATCH SET
			   d.name = $name, d.type = $type, d.type_cn = $typeCN,
			   d.vendor = $vendor, d.vendor_cn = $vendorCN, d.mip = $mip,
			   d.updated_at = $now`,
			map[string]any{
				"id": d.id, "name": d.name, "type": d.typ, "typeCN": d.typeCN,
				"vendor": d.vendor, "vendorCN": d.vendorCN, "mip": d.mip, "now": now,
			},
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(dbName),
		)
		if err != nil {
			return fmt.Errorf("merge device %s: %w", d.id, err)
		}
	}

	// 写入 CONNECTED_TO 关系
	for _, conn := range connSeeds {
		relID := uuid.NewSHA1(uuid.Nil, []byte(conn[0]+"-"+conn[1])).String()
		_, err := neo4j.ExecuteQuery(ctx, driver,
			`MATCH (a:Device {id: $from}), (b:Device {id: $to})
			 MERGE (a)-[r:CONNECTED_TO {id: $relID}]->(b)
			 ON CREATE SET r.created_at = $now`,
			map[string]any{"from": conn[0], "to": conn[1], "relID": relID, "now": now},
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(dbName),
		)
		if err != nil {
			return fmt.Errorf("merge connection %s->%s: %w", conn[0], conn[1], err)
		}
	}
	return nil
}
