package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"matrix/api/internal/service"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login POST /api/v1/auth/login
// 验证账号密码，签发 JWT 并在响应体中返回。
// cookie 由 Next.js 服务端负责写入，Go API 不直接操作客户端 cookie。
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.Auth.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

type larkExchangeRequest struct {
	Code string `json:"code" binding:"required"`
}

// LarkExchange POST /internal/auth/lark/exchange
// 仅供 Next.js 服务端调用（通过 X-Internal-Secret 鉴权）。
// 用飞书 OAuth code 换取系统 JWT。白名单制：未授权用户返回 403。
func (h *Handler) LarkExchange(c *gin.Context) {
	if c.GetHeader("X-Internal-Secret") != h.internalSecret {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var req larkExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.Auth.LarkExchange(c.Request.Context(), req.Code)
	if err != nil {
		// 不在白名单：明确返回 403，前端据此提示"无权限"
		if errors.Is(err, service.ErrNotAuthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "lark auth failed"})
		return
	}
	c.JSON(http.StatusOK, resp)
}
