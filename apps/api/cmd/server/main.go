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
)

func main() {
	encryptVal := flag.String("encrypt", "", "加密一个明文值并输出结果，可直接填入配置文件")
	initAdminPwd := flag.String("init-admin-password", "", "重置 admin 账号密码后退出，不启动服务")
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

	// -init-admin-password 模式：重置 admin 密码后退出
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
		fmt.Println("admin 账号密码已更新")
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
	svcs := service.New(repos, cfg.JWT)
	h := handler.New(svcs, repos, cfg.JWT)

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Stop(ctx); err != nil {
		log.Logger.Errorf("stop server: %v", err)
	}
	log.Logger.Info("server exited")
}
