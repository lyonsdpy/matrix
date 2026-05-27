// sync 是飞书通讯录全量同步工具：拉取全公司用户写入 Neo4j User 节点。
// MERGE 以 feishu_id（open_id）为键保证幂等，重复执行安全。
//
// 用法：
//
//	go run ./cmd/sync
//	go run ./cmd/sync -config configs/config.yaml
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"

	"matrix/api/internal/infra/lark"
	neo4jdb "matrix/api/internal/infra/neo4j"
	"matrix/api/pkg/config"
	"matrix/api/pkg/log"
)

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

	if cfg.Lark.AppID == "" || cfg.Lark.AppSecret == "" {
		log.Logger.Fatal("lark.app_id / app_secret 未配置，无法同步通讯录")
	}

	ctx := context.Background()

	// 飞书 client + 通讯录拉取器
	client, err := lark.NewClient(lark.Config{AppID: cfg.Lark.AppID, AppSecret: cfg.Lark.AppSecret})
	if err != nil {
		log.Logger.Fatalf("create lark client: %v", err)
	}
	fetcher := lark.NewContactFetcher(client, log.Log)

	// Neo4j 连接
	driver, err := neo4jdb.Open(cfg.Neo4j)
	if err != nil {
		log.Logger.Fatalf("connect neo4j: %v", err)
	}
	if driver == nil {
		log.Logger.Fatal("neo4j URI 未配置，无法写入 User 节点")
	}
	defer driver.Close(ctx)

	if err := neo4jdb.InitSchema(ctx, driver, cfg.Neo4j.Database); err != nil {
		log.Logger.Fatalf("neo4j init schema: %v", err)
	}

	// 拉取全量通讯录用户
	log.Logger.Info("开始拉取飞书通讯录...")
	users, err := fetcher.FetchAllUsers(ctx)
	if err != nil {
		log.Logger.Fatalf("fetch all users: %v", err)
	}
	log.Logger.Infof("拉取到 %d 个飞书用户，开始写入 Neo4j", len(users))

	// 逐个 MERGE 写入 Neo4j User 节点
	var written, skipped int
	for _, u := range users {
		if u.OpenID == "" {
			// 没有 open_id 的用户无法与扫码登录关联，跳过
			log.Logger.Warnf("用户 %s（user_id=%s）缺少 open_id，跳过", u.Name, u.UserID)
			skipped++
			continue
		}
		if err := upsertUser(ctx, driver, cfg.Neo4j.Database, u); err != nil {
			log.Logger.Errorf("写入用户 %s（open_id=%s）失败: %v", u.Name, u.OpenID, err)
			skipped++
			continue
		}
		written++
	}

	log.Logger.Infof("✓ 同步完成：写入 %d，跳过 %d", written, skipped)
}

// upsertUser 以 feishu_id（open_id）为业务唯一键 MERGE User 节点。
// id 由 open_id 派生（SHA1）保证幂等；name 等飞书属性每次同步刷新。
func upsertUser(ctx context.Context, driver neo4j.Driver, dbName string, u lark.User) error {
	now := time.Now().UTC()
	_, err := neo4j.ExecuteQuery(ctx, driver,
		`MERGE (u:User {feishu_id: $openID})
		 ON CREATE SET
		   u.id = $id, u.created_at = $now
		 SET
		   u.name = $name, u.user_id = $userID, u.email = $email,
		   u.mobile = $mobile, u.status = $status, u.department_ids = $deptIDs,
		   u.updated_at = $now`,
		map[string]any{
			"openID":  u.OpenID,
			"id":      deriveID(u.OpenID),
			"name":    u.Name,
			"userID":  u.UserID,
			"email":   u.Email,
			"mobile":  u.Mobile,
			"status":  int64(u.Status),
			"deptIDs": u.DepartmentIDs,
			"now":     now,
		},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(dbName),
	)
	return err
}

// deriveID 由 open_id 派生稳定的节点 id，与 Neo4jUserGraphRepo.CreateUser 保持一致。
func deriveID(openID string) string {
	return uuid.NewSHA1(uuid.Nil, []byte(openID)).String()
}
