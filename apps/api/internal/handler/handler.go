package handler

import (
	"github.com/gin-gonic/gin"
	"matrix/api/internal/service"
)

type Handler struct {
	svc *service.Services
}

func New(svc *service.Services) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", h.Health)

	v1 := r.Group("/api/v1")
	{
		_ = v1 // mount v1 routes here
	}
}
