// sync 是飞书全量同步工具：依次跑通讯录和终端两个独立同步任务，与前端两个独立按钮共用一套逻辑。
//
// 用法：
//
//	go run ./cmd/sync                      # 通讯录 + 终端 都跑
//	go run ./cmd/sync -only=contacts       # 仅通讯录
//	go run ./cmd/sync -only=endpoints      # 仅终端
//	go run ./cmd/sync -config configs/config.yaml
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"matrix/api/internal/infra/lark"
	neo4jdb "matrix/api/internal/infra/neo4j"
	"matrix/api/internal/repository/neo4j_repo"
	"matrix/api/internal/service"
	"matrix/api/pkg/config"
	"matrix/api/pkg/log"
)

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	only := flag.String("only", "", "限定阶段：contacts | endpoints；留空表示两个都跑")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	log.Config = &cfg.Log
	log.Config.Reset()

	if cfg.Lark.AppID == "" || cfg.Lark.AppSecret == "" {
		log.Logger.Fatal("lark.app_id / app_secret 未配置，无法同步")
	}

	ctx := context.Background()

	client, err := lark.NewClient(lark.Config{AppID: cfg.Lark.AppID, AppSecret: cfg.Lark.AppSecret})
	if err != nil {
		log.Logger.Fatalf("create lark client: %v", err)
	}
	fetcher := lark.NewContactFetcher(client, log.Log)
	deviceFetcher := lark.NewDeviceFetcher(client, log.Log)

	driver, err := neo4jdb.Open(cfg.Neo4j)
	if err != nil {
		log.Logger.Fatalf("connect neo4j: %v", err)
	}
	if driver == nil {
		log.Logger.Fatal("neo4j URI 未配置，无法写入节点")
	}
	defer driver.Close(ctx)

	if err := neo4jdb.InitSchema(ctx, driver, cfg.Neo4j.Database); err != nil {
		log.Logger.Fatalf("neo4j init schema: %v", err)
	}

	deptRepo := neo4j_repo.NewNeo4jDepartmentRepo(driver, cfg.Neo4j.Database)
	userRepo := neo4j_repo.NewNeo4jUserGraphRepo(driver, cfg.Neo4j.Database)
	endpointRepo := neo4j_repo.NewNeo4jEndpointRepo(driver, cfg.Neo4j.Database)

	if *only == "" || *only == "contacts" {
		runContacts(ctx, fetcher, userRepo, deptRepo)
	}
	if *only == "" || *only == "endpoints" {
		runEndpoints(ctx, deviceFetcher, endpointRepo)
	}
}

func runContacts(ctx context.Context, fetcher lark.ContactFetcher, userRepo *neo4j_repo.Neo4jUserGraphRepo, deptRepo *neo4j_repo.Neo4jDepartmentRepo) {
	svc := service.NewSyncService(fetcher, userRepo, deptRepo)
	jobID, started, err := svc.Start(ctx)
	if err != nil {
		log.Logger.Fatalf("start contacts sync: %v", err)
	}
	if !started {
		log.Logger.Warnf("contacts sync 已在运行（job=%s），等待其完成", jobID)
	} else {
		log.Logger.Infof("contacts sync job %s started", jobID)
	}
	lastPhase := service.SyncPhase("")
	for {
		snap := svc.Snapshot()
		if snap.Phase != lastPhase {
			log.Logger.Infof("[contacts] phase=%s done=%d total=%d", snap.Phase, snap.Done, snap.Total)
			lastPhase = snap.Phase
		}
		if snap.Phase == service.PhaseDone || snap.Phase == service.PhaseFailed {
			if snap.Phase == service.PhaseFailed {
				log.Logger.Fatalf("contacts sync failed: %s", snap.Error)
			}
			log.Logger.Infof("✓ 通讯录同步完成 用时 %.1fs", float64(snap.DurationMs)/1000)
			log.Logger.Infof("部门：新增 %d，更新 %d，删除 %d，总计 %d",
				snap.Departments.Created, snap.Departments.Updated, snap.Departments.Deleted, snap.Departments.Total)
			log.Logger.Infof("用户：新增 %d，更新 %d，删除 %d，总计 %d",
				snap.Users.Created, snap.Users.Updated, snap.Users.Deleted, snap.Users.Total)
			return
		}
		time.Sleep(2 * time.Second)
	}
}

func runEndpoints(ctx context.Context, fetcher lark.DeviceFetcher, repo *neo4j_repo.Neo4jEndpointRepo) {
	svc := service.NewEndpointSyncService(fetcher, repo)
	jobID, started, err := svc.Start(ctx)
	if err != nil {
		log.Logger.Fatalf("start endpoint sync: %v", err)
	}
	if !started {
		log.Logger.Warnf("endpoint sync 已在运行（job=%s），等待其完成", jobID)
	} else {
		log.Logger.Infof("endpoint sync job %s started", jobID)
	}
	lastPhase := service.EndpointSyncPhase("")
	for {
		snap := svc.Snapshot()
		if snap.Phase != lastPhase {
			log.Logger.Infof("[endpoints] phase=%s done=%d total=%d", snap.Phase, snap.Done, snap.Total)
			lastPhase = snap.Phase
		}
		if snap.Phase == service.EndpointPhaseDone || snap.Phase == service.EndpointPhaseFailed {
			if snap.Phase == service.EndpointPhaseFailed {
				log.Logger.Fatalf("endpoint sync failed: %s", snap.Error)
			}
			log.Logger.Infof("✓ 终端同步完成 用时 %.1fs", float64(snap.DurationMs)/1000)
			log.Logger.Infof("终端：新增 %d，更新 %d，删除 %d，总计 %d",
				snap.Endpoints.Created, snap.Endpoints.Updated, snap.Endpoints.Deleted, snap.Endpoints.Total)
			return
		}
		time.Sleep(2 * time.Second)
	}
}
