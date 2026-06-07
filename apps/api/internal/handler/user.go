package handler

import (
	"net/http"
	"strconv"

	"matrix/api/pkg/auth"
	"matrix/api/pkg/errno"

	"github.com/gin-gonic/gin"
)

// GetUserDetail GET /api/v1/users/:id
// 通讯录-用户详情：基本信息 + 所属部门(带完整路径) + 上级领导 + 管理部门。:id 为飞书 open_id (feishu_id)。
func (h *Handler) GetUserDetail(c *gin.Context) {
	openID := c.Param("id")
	if openID == "" {
		c.JSON(http.StatusBadRequest, errno.New(400, "missing user id"))
		return
	}
	detail, err := h.svc.User.GetDetail(c.Request.Context(), openID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询用户失败").WithErr(err))
		return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, errno.New(404, "用户不存在"))
		return
	}
	c.JSON(http.StatusOK, detail)
}

// ListUsers GET /api/v1/users
// 查询已同步的飞书用户，支持 ?search= 模糊搜索、?cursor= 游标分页、?limit= 每页数量。
func (h *Handler) ListUsers(c *gin.Context) {
	search := c.Query("search")
	cursor := c.Query("cursor")
	limit, _ := strconv.Atoi(c.Query("limit"))

	list, err := h.svc.User.ListSynced(c.Request.Context(), search, cursor, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询用户失败").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, list)
}

// GetAuthUserRoles GET /api/v1/auth-users/:id/roles
// 用户管理页授权弹窗预填用：取某本地账号当前绑定的角色集合。
// :id 是 users.id（UUID）——/auth-users 前缀明确区分系统本地账号 vs 飞书通讯录用户(/users)
func (h *Handler) GetAuthUserRoles(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, errno.New(400, "missing user id"))
		return
	}
	roles, err := h.svc.Role.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询用户角色失败").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": roles})
}

type setUserRolesRequest struct {
	RoleIDs []string `json:"role_ids"` // 空切片代表清空（撤销所有授权）
}

// SetAuthUserRoles PUT /api/v1/auth-users/:id/roles
// 覆盖式：传入的 role_ids 就是新的完整集合，未列出的被撤销
func (h *Handler) SetAuthUserRoles(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, errno.New(400, "missing user id"))
		return
	}
	var req setUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errno.New(400, err.Error()))
		return
	}
	grantedBy := ""
	if u := auth.FromContext(c.Request.Context()); u != nil {
		grantedBy = u.ID
	}
	if err := h.svc.Role.SetUserRoles(c.Request.Context(), userID, req.RoleIDs, grantedBy); err != nil {
		writeRoleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetAuthUserByLark GET /api/v1/auth-users/by-lark/:openId
// 通过飞书 open_id 反查系统本地账号；未授权（不在白名单）返回 404
// 前端通讯录用户详情抽屉用：判断该飞书用户是否能登录系统、用 user_id 调授权接口
func (h *Handler) GetAuthUserByLark(c *gin.Context) {
	openID := c.Param("openId")
	if openID == "" {
		c.JSON(http.StatusBadRequest, errno.New(400, "missing open id"))
		return
	}
	user, err := h.svc.Auth.FindByLarkOpenID(c.Request.Context(), openID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询失败").WithErr(err))
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, errno.New(404, "用户未在白名单"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"lark_open_id": user.LarkOpenID,
	})
}
