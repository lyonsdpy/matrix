package handler

import (
	"net/http"

	"matrix/api/pkg/auth"
	"matrix/api/pkg/errno"

	"github.com/gin-gonic/gin"
)

// ListPermissions GET /api/v1/permissions
// 列出系统所有权限码（前端构建权限树用）
func (h *Handler) ListPermissions(c *gin.Context) {
	list, err := h.svc.Permission.ListAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询权限码失败").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list})
}

// MyPermissions GET /api/v1/me/permissions
// 当前用户拥有的所有权限码集合，给前端按钮/菜单显隐用。
// admin 角色用户返回全部权限码并带 is_admin=true 标志，前端可据此特殊渲染。
func (h *Handler) MyPermissions(c *gin.Context) {
	u := auth.FromContext(c.Request.Context())
	if u == nil {
		c.JSON(http.StatusUnauthorized, errno.New(401, "not authenticated"))
		return
	}
	codes, isAdmin, err := h.svc.Permission.MyPermissions(c.Request.Context(), u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询权限失败").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id":     u.ID,
		"username":    u.Username,
		"is_admin":    isAdmin,
		"permissions": codes,
	})
}
