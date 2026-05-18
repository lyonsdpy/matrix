package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"matrix/api/internal/model"
	"matrix/api/pkg/errno"
)

// Hello GET /api/v1/hello?name=xxx
// 完整服务链演示：handler → service → repository
func (h *Handler) Hello(c *gin.Context) {
	// 1. 绑定并校验请求参数（binding:"required" 失败时 gin 自动返回 400）
	var req model.HelloRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, errno.New(400, "参数错误").WithErr(err))
		return
	}

	// 2. 调用 Service 层执行业务逻辑
	msg := h.svc.Hello.SayHello(req.Name)

	// 3. 返回标准 JSON 响应
	c.JSON(http.StatusOK, model.HelloResponse{Message: msg})
}
