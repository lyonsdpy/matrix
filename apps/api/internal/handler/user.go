package handler

import (
	"net/http"
	"strconv"

	"matrix/api/pkg/errno"

	"github.com/gin-gonic/gin"
)

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
