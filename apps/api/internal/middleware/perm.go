package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"matrix/api/pkg/auth"
)

// PermissionChecker 中间件最小依赖；具体由 service.PermissionService 实现。
// 在使用方定义接口，避免 middleware 包反向依赖 service 包。
type PermissionChecker interface {
	UserHasAny(ctx context.Context, userID string, codes []string) (bool, error)
}

// RequirePermission 路由级权限校验中间件。
// 用法：v1.GET("/endpoints", middleware.RequirePermission(svc, "endpoint:read"), h.ListEndpoints)
// 多个 code 任一命中即放行（用于一个接口允许多个角色访问的场景）。
// admin 角色特判已内置在 svc.UserHasAny 中（B 方案），无需在此处显式判断。
func RequirePermission(svc PermissionChecker, codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := auth.FromContext(c.Request.Context())
		if u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		ok, err := svc.UserHasAny(c.Request.Context(), u.ID, codes)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
			return
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "forbidden",
				"need":  codes,
			})
			return
		}
		c.Next()
	}
}
