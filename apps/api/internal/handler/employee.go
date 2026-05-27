package handler

import (
	"errors"
	"net/http"

	"matrix/api/internal/service"
	"matrix/api/pkg/errno"

	"github.com/gin-gonic/gin"
)

// GetEmployee GET /api/v1/employees/:id
// 演示跨源聚合接口：handler 只做参数绑定和响应格式化，业务 join 在 service 层完成。
func (h *Handler) GetEmployee(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, errno.New(400, "missing employee id"))
		return
	}

	view, err := h.svc.Employee.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, errno.New(404, "员工不存在").WithErr(err))
			return
		}
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询员工失败").WithErr(err))
		return
	}

	c.JSON(http.StatusOK, view)
}
