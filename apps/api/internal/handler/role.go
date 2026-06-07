package handler

import (
	"errors"
	"net/http"
	"strconv"

	"matrix/api/internal/service"
	"matrix/api/pkg/auth"
	"matrix/api/pkg/errno"

	"github.com/gin-gonic/gin"
)

// ListRoles GET /api/v1/roles
func (h *Handler) ListRoles(c *gin.Context) {
	list, err := h.svc.Role.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询角色失败").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list})
}

// GetRole GET /api/v1/roles/:id
// 详情：基本信息 + 已绑权限码集合
func (h *Handler) GetRole(c *gin.Context) {
	id := c.Param("id")
	detail, err := h.svc.Role.Get(c.Request.Context(), id)
	if err != nil {
		writeRoleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

type createRoleRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// CreateRole POST /api/v1/roles
func (h *Handler) CreateRole(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errno.New(400, err.Error()))
		return
	}
	r, err := h.svc.Role.Create(c.Request.Context(), req.Code, req.Name, req.Description)
	if err != nil {
		writeRoleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

type updateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateRole PUT /api/v1/roles/:id
// code 不可改（机器名一旦确定不允许变更，避免破坏已发出的绑定引用）
func (h *Handler) UpdateRole(c *gin.Context) {
	id := c.Param("id")
	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errno.New(400, err.Error()))
		return
	}
	r, err := h.svc.Role.Update(c.Request.Context(), id, req.Name, req.Description)
	if err != nil {
		writeRoleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

// DeleteRole DELETE /api/v1/roles/:id
func (h *Handler) DeleteRole(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Role.Delete(c.Request.Context(), id); err != nil {
		writeRoleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type setRolePermissionsRequest struct {
	Codes []string `json:"codes"` // 空切片代表清空
}

// SetRolePermissions PUT /api/v1/roles/:id/permissions
// 覆盖式：传入的 codes 就是新的完整集合，未列出的被解除
func (h *Handler) SetRolePermissions(c *gin.Context) {
	id := c.Param("id")
	var req setRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errno.New(400, err.Error()))
		return
	}
	if err := h.svc.Role.SetPermissions(c.Request.Context(), id, req.Codes); err != nil {
		writeRoleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ListRoleUsers GET /api/v1/roles/:id/users
// 角色管理页右栏：该角色下的用户列表
func (h *Handler) ListRoleUsers(c *gin.Context) {
	id := c.Param("id")
	limit, _ := strconv.Atoi(c.Query("limit"))
	users, err := h.svc.Role.ListUsers(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询角色用户失败").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": users})
}

type addRoleUsersRequest struct {
	// 二选一传入（同时给定时 lark_open_ids 优先）
	UserIDs      []string `json:"user_ids"`
	LarkOpenIDs  []string `json:"lark_open_ids"`
}

// AddRoleUsers POST /api/v1/roles/:id/users
// 批量给角色添加用户。granted_by 自动取当前登录者。
// 前端通常传 lark_open_ids（从飞书搜索拿到的就是 open_id）；
// user_ids 入口保留给后续"用户授权管理页"场景
func (h *Handler) AddRoleUsers(c *gin.Context) {
	id := c.Param("id")
	var req addRoleUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errno.New(400, err.Error()))
		return
	}
	grantedBy := ""
	if u := auth.FromContext(c.Request.Context()); u != nil {
		grantedBy = u.ID
	}
	if len(req.LarkOpenIDs) > 0 {
		n, err := h.svc.Role.AddUsersByLarkOpenIDs(c.Request.Context(), id, req.LarkOpenIDs, grantedBy)
		if err != nil {
			writeRoleErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"ok":      true,
			"added":   n,
			"skipped": len(req.LarkOpenIDs) - n, // 不在白名单的飞书用户
		})
		return
	}
	if err := h.svc.Role.AddUsers(c.Request.Context(), id, req.UserIDs, grantedBy); err != nil {
		writeRoleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "added": len(req.UserIDs)})
}

// RemoveRoleUser DELETE /api/v1/roles/:id/users/:userId
func (h *Handler) RemoveRoleUser(c *gin.Context) {
	id := c.Param("id")
	userID := c.Param("userId")
	if err := h.svc.Role.RemoveUser(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "解绑用户失败").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// writeRoleErr 统一映射 RoleService 错误到 HTTP 状态码
func writeRoleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrRoleNotFound):
		c.JSON(http.StatusNotFound, errno.New(404, "角色不存在"))
	case errors.Is(err, service.ErrRoleCodeConflict):
		c.JSON(http.StatusConflict, errno.New(409, "角色 code 已存在"))
	case errors.Is(err, service.ErrSystemRoleProtected):
		c.JSON(http.StatusForbidden, errno.New(403, "系统内置角色受保护"))
	case errors.Is(err, service.ErrInvalidRoleCode):
		c.JSON(http.StatusBadRequest, errno.New(400, "角色 code 不合法（仅允许小写字母/数字/下划线，2-64 字符）"))
	case errors.Is(err, service.ErrUnknownPermissionCode):
		c.JSON(http.StatusBadRequest, errno.New(400, err.Error()))
	default:
		c.JSON(http.StatusInternalServerError, errno.New(500, "操作失败").WithErr(err))
	}
}
