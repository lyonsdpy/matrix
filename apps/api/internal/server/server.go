package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"matrix/api/internal/handler"
	"matrix/api/internal/middleware"
	"matrix/api/pkg/config"
	"matrix/api/pkg/log"
)

type Server struct {
	http *http.Server
	cfg  *config.Config
	h    *handler.Handler
}

func New(cfg *config.Config, h *handler.Handler) *Server {
	return &Server{cfg: cfg, h: h}
}

func (s *Server) Init() error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.Logger(), middleware.Recovery(), middleware.Cors())
	s.h.Register(r)

	s.http = &http.Server{
		Addr:         s.cfg.Server.Addr,
		Handler:      r,
		ReadTimeout:  time.Duration(s.cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.cfg.Server.WriteTimeout) * time.Second,
	}
	return nil
}

func (s *Server) Start() error {
	log.Logger.Infof("listening on %s", s.cfg.Server.Addr)
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Logger.Info("shutting down...")
	return s.http.Shutdown(ctx)
}
