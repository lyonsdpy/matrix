package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"matrix/api/internal/handler"
	neo4jdb "matrix/api/internal/infra/neo4j"
	"matrix/api/internal/infra/postgres"
	"matrix/api/internal/repository"
	"matrix/api/internal/repository/pg_repo"
	"matrix/api/internal/server"
	"matrix/api/internal/service"
	"matrix/api/pkg/auth"
	"matrix/api/pkg/config"
	"matrix/api/pkg/crypto"
	"matrix/api/pkg/log"
	"matrix/api/pkg/perm"
)

func main() {
	encryptVal := flag.String("encrypt", "", "加密一个明文值并输出结果，可直接填入配置文件")
	initAdminPwd := flag.String("init-admin-password", "", "重置 admin 账号密码后退出，不启动服务（自带解锁：清空该账号所有登录失败计数）")
	unlockAdmin := flag.Bool("unlock-admin", false, "清空 admin 账号的所有登录失败计数与软锁定后退出，不启动服务")
	flag.Parse()

	// -encrypt 模式：输出加密结果后退出，不启动服务
	if *encryptVal != "" {
		encrypted, err := crypto.Encrypt(*encryptVal)
		if err != nil {
			fmt.Fprintf(os.Stderr, "encrypt error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(encrypted)
		os.Exit(0)
	}

	// -init-admin-password 模式：重置 admin 密码后退出（自带解锁）
	if *initAdminPwd != "" {
		cfg, err := config.Load("configs/config.yaml")
		if err != nil {
			fmt.Fprintf(os.Stderr, "load config: %v\n", err)
			os.Exit(1)
		}
		db, err := postgres.Open(cfg.Postgres.DSN())
		if err != nil {
			fmt.Fprintf(os.Stderr, "connect postgres: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()
		if err := postgres.RunMigrationsEmbedded(cfg.Postgres.DSN()); err != nil {
			fmt.Fprintf(os.Stderr, "run migrations: %v\n", err)
			os.Exit(1)
		}
		hash, err := auth.HashPassword(*initAdminPwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hash password: %v\n", err)
			os.Exit(1)
		}
		userRepo := pg_repo.NewAuthUserRepository(db)
		if err := userRepo.UpdatePassword(context.Background(), "admin", hash); err != nil {
			fmt.Fprintf(os.Stderr, "update password: %v\n", err)
			os.Exit(1)
		}
		// 改密同时解锁：避免运维改了密码却忘了 admin 仍在软锁中
		laRepo := pg_repo.NewLoginAttemptRepository(db)
		cleared, _ := laRepo.ResetByUsername(context.Background(), "admin")
		fmt.Printf("admin 账号密码已更新（同时清理 %d 条登录失败记录）\n", cleared)
		os.Exit(0)
	}

	// -unlock-admin 模式：清空 admin 的所有登录失败计数与软锁定
	if *unlockAdmin {
		cfg, err := config.Load("configs/config.yaml")
		if err != nil {
			fmt.Fprintf(os.Stderr, "load config: %v\n", err)
			os.Exit(1)
		}
		db, err := postgres.Open(cfg.Postgres.DSN())
		if err != nil {
			fmt.Fprintf(os.Stderr, "connect postgres: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()
		if err := postgres.RunMigrationsEmbedded(cfg.Postgres.DSN()); err != nil {
			fmt.Fprintf(os.Stderr, "run migrations: %v\n", err)
			os.Exit(1)
		}
		laRepo := pg_repo.NewLoginAttemptRepository(db)
		cleared, err := laRepo.ResetByUsername(context.Background(), "admin")
		if err != nil {
			fmt.Fprintf(os.Stderr, "unlock admin: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("admin 账号已解锁（清理 %d 条登录失败记录）\n", cleared)
		os.Exit(0)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Logger.Fatalf("load config: %v", err)
	}
	log.Config = &cfg.Log
	log.Config.Reset()

	// PostgreSQL 连接（SyncRepos 所需）
	db, err := postgres.Open(cfg.Postgres.DSN())
	if err != nil {
		log.Logger.Fatalf("connect postgres: %v", err)
	}
	if err := postgres.RunMigrationsEmbedded(cfg.Postgres.DSN()); err != nil {
		log.Logger.Fatalf("run postgres migrations: %v", err)
	}
	log.Logger.Info("postgres migrations applied")

	// 权限码权威列表同步入库（代码维护、UI 只读；新增/改名/删除均会反映到 permissions 表）
	if err := perm.SyncCatalog(context.Background(), db); err != nil {
		log.Logger.Fatalf("sync permission catalog: %v", err)
	}
	log.Logger.Infof("permission catalog synced: %d codes", len(perm.Codes))

	// Neo4j 连接（URI 为空时自动降级为内存存根）
	neo4jDriver, err := neo4jdb.Open(cfg.Neo4j)
	if err != nil {
		log.Logger.Fatalf("connect neo4j: %v", err)
	}
	if neo4jDriver != nil {
		defer neo4jDriver.Close(context.Background())
		log.Logger.Infof("neo4j connected: %s", cfg.Neo4j.URI)
		if err := neo4jdb.InitSchema(context.Background(), neo4jDriver, cfg.Neo4j.Database); err != nil {
			log.Logger.Fatalf("neo4j init schema: %v", err)
		}
		log.Logger.Info("neo4j schema initialized")
	} else {
		log.Logger.Warn("neo4j not configured, using in-memory stub")
	}

	repos := repository.New(db, neo4jDriver, cfg.Neo4j.Database)
	svcs := service.New(repos, cfg.JWT, cfg.Lark)
	h := handler.New(svcs, repos.Graph.Device, cfg.JWT, cfg.Lark)

	srv := server.New(cfg, h)
	if err := srv.Init(); err != nil {
		log.Logger.Fatalf("init server: %v", err)
	}

	go func() {
		if err := srv.Start(); err != nil {
			log.Logger.Fatalf("start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 先停止后台 goroutine，再等待 HTTP 连接排尽
	svcs.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Stop(ctx); err != nil {
		log.Logger.Errorf("stop server: %v", err)
	}
	log.Logger.Info("server exited")
}
