package handler

import (
	"net/http"

	"matrix/api/pkg/errno"

	"github.com/gin-gonic/gin"
)

// GetDepartmentDetail GET /api/v1/departments/:id
// 通讯录-部门详情：基本信息 + 路径 + 子部门 + 直属成员前 N + 递归人数。
func (h *Handler) GetDepartmentDetail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, errno.New(400, "missing department id"))
		return
	}
	detail, err := h.svc.Department.GetDetail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询部门失败").WithErr(err))
		return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, errno.New(404, "部门不存在"))
		return
	}
	c.JSON(http.StatusOK, detail)
}
