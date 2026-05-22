package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"matrix/api/internal/middleware"
	"matrix/api/internal/service"
	"matrix/api/pkg/crypto"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login POST /api/v1/auth/login
// 验证账号密码，成功后将 JWT 写入 HttpOnly cookie，body 只返回用户信息。
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

	// AES-256-GCM 加密 JWT 后再写入 cookie，客户端无法按 JWT 格式解析 claims
	encrypted, err := crypto.Encrypt(resp.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	maxAge := int(time.Until(resp.ExpiresAt).Seconds())
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(middleware.SessionCookie, encrypted, maxAge, "/", "", h.secureCookie, true)

	// body 不暴露 token，只返回用户信息，前端用来更新 UI 状态
	c.JSON(http.StatusOK, gin.H{
		"user":       resp.User,
		"expires_at": resp.ExpiresAt,
	})
}

// Logout POST /api/v1/auth/logout
// 清除 session cookie，客户端即退出登录。
func (h *Handler) Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(middleware.SessionCookie, "", -1, "/", "", h.secureCookie, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
