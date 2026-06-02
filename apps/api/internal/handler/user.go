package handler

import (
	"net/http"
	"strconv"

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
