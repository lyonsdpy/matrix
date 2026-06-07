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
	"matrix/api/pkg/perm"
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

	// 权限码权威列表同步（seed 也跑一次，保证 role_permissions 外键 ok）
	if err := perm.SyncCatalog(ctx, db); err != nil {
		log.Logger.Fatalf("sync permission catalog: %v", err)
	}
	log.Logger.Infof("permission catalog synced: %d codes", len(perm.Codes))

	if err := seedPostgres(ctx, db); err != nil {
		log.Logger.Fatalf("seed postgres: %v", err)
	}
	log.Logger.Infof("postgres: seeded %d departments, %d employees, admin user, roles", len(deptSeeds), len(empSeeds))

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
	if err := seedRoles(ctx, db); err != nil {
		return fmt.Errorf("seed roles: %w", err)
	}
	if err := seedUserRoles(ctx, db); err != nil {
		return fmt.Errorf("seed user roles: %w", err)
	}
	return nil
}

// seedRoles 写入系统内置角色 admin / viewer。
// admin 走中间件特判（不绑权限），viewer 绑全部 kind=page 权限码做"只读"角色。
func seedRoles(ctx context.Context, db *sqlx.DB) error {
	// 内置角色，is_system=true 不可删；ON CONFLICT 保证重跑幂等
	roleSeeds := []struct {
		code, name, desc string
	}{
		{perm.RoleCodeAdmin, "系统管理员", "拥有所有权限，受中间件特判绕过权限校验"},
		{perm.RoleCodeViewer, "只读角色", "可查看所有页面，无操作权限"},
	}
	for _, r := range roleSeeds {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO roles (code, name, description, is_system)
			 VALUES ($1, $2, $3, TRUE)
			 ON CONFLICT (code) DO UPDATE
			   SET name = EXCLUDED.name, description = EXCLUDED.description, is_system = TRUE`,
			r.code, r.name, r.desc,
		); err != nil {
			return fmt.Errorf("upsert role %s: %w", r.code, err)
		}
	}

	// viewer 角色：覆盖式绑定所有 kind=page 的权限码
	// 先清空 viewer 已有绑定，再写入当前所有 page 码（随权限码增删自动跟进）
	var viewerID string
	if err := db.GetContext(ctx, &viewerID,
		`SELECT id FROM roles WHERE code = $1`, perm.RoleCodeViewer,
	); err != nil {
		return fmt.Errorf("lookup viewer role: %w", err)
	}
	if _, err := db.ExecContext(ctx,
		`DELETE FROM role_permissions WHERE role_id = $1`, viewerID,
	); err != nil {
		return fmt.Errorf("clear viewer permissions: %w", err)
	}
	for _, p := range perm.Codes {
		if p.Kind != perm.KindPage {
			continue
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO role_permissions (role_id, permission_code) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`,
			viewerID, p.Code,
		); err != nil {
			return fmt.Errorf("grant %s to viewer: %w", p.Code, err)
		}
	}
	log.Logger.Info("roles seeded: admin (中间件特判), viewer (所有 page 权限)")
	return nil
}

// seedUserRoles 给本地 admin 账号 + 已存在的 daipengyuan 绑 admin 角色。
// 幂等：用户不存在则跳过；ON CONFLICT 不重复插入。
func seedUserRoles(ctx context.Context, db *sqlx.DB) error {
	var adminRoleID string
	if err := db.GetContext(ctx, &adminRoleID,
		`SELECT id FROM roles WHERE code = $1`, perm.RoleCodeAdmin,
	); err != nil {
		return fmt.Errorf("lookup admin role: %w", err)
	}

	// 给本地 admin 账号绑 admin 角色
	if _, err := db.ExecContext(ctx,
		`INSERT INTO user_roles (user_id, role_id)
		 SELECT id, $1 FROM users WHERE username = 'admin'
		 ON CONFLICT DO NOTHING`,
		adminRoleID,
	); err != nil {
		return fmt.Errorf("bind admin user to admin role: %w", err)
	}

	// daipengyuan（戴澎源）：飞书 open_id 预设管理员
	// 用户尚未通过 lark 登录入库则跳过，下次重跑 seed 即补绑
	const daipengyuanOpenID = "ou_df061cdb2f20ffaaedd566e672a43261"
	res, err := db.ExecContext(ctx,
		`INSERT INTO user_roles (user_id, role_id)
		 SELECT id, $1 FROM users WHERE lark_open_id = $2
		 ON CONFLICT DO NOTHING`,
		adminRoleID, daipengyuanOpenID,
	)
	if err != nil {
		return fmt.Errorf("bind daipengyuan to admin role: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		log.Logger.Info("daipengyuan 用户尚未入库（首次飞书登录后重跑 seed 即可绑 admin）")
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
