package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"matrix/api/internal/handler"
	"matrix/api/internal/infra/postgres"
	"matrix/api/internal/repository"
	"matrix/api/internal/server"
	"matrix/api/internal/service"
	"matrix/api/pkg/config"
	"matrix/api/pkg/log"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Logger.Fatalf("load config: %v", err)
	}
	log.Config = &cfg.Log
	log.Config.Reset()

	// PostgreSQL 连接（SyncRepos 所需）
	db, err := postgres.Open(cfg.Postgres.DSN)
	if err != nil {
		log.Logger.Fatalf("connect postgres: %v", err)
	}

	repos := repository.New(db)
	svcs := service.New(repos)
	h := handler.New(svcs, repos)

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
