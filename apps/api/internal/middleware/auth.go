package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"matrix/api/pkg/auth"
	"matrix/api/pkg/crypto"
)

const SessionCookie = "matrix_session"

// Auth 从 HttpOnly cookie 中取出加密 session，先 AES 解密再验证 JWT。
// cookie 缺失、解密失败、JWT 非法均返回 401。
// 客户端看到的 cookie 值是密文，无法按 JWT 格式解析出 claims。
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		encrypted, err := c.Cookie(SessionCookie)
		if err != nil || encrypted == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		tokenStr, err := crypto.Decrypt(encrypted)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
			return
		}
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
