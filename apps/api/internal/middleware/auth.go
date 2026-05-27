package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"matrix/api/pkg/auth"
)

// Auth 从 Authorization: Bearer <token> 中提取并验证 JWT。
// 所有 cookie 管理由 Next.js 服务端负责；Go API 只处理明文 JWT。
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ParseToken(secret, tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
			return
		}
		ctx := auth.WithUser(c.Request.Context(), &auth.User{
			ID:       claims.UserID,
			Username: claims.Username,
			Roles:    claims.Roles,
		})
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
