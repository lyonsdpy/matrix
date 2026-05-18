package handler

import (
	"matrix/api/internal/service"

	"github.com/gin-gonic/gin"
)

// Handler 聚合所有 HTTP 处理方法，持有 Services 引用。
// 职责：参数绑定、调用 Service、返回 HTTP 响应，不含业务逻辑。
type Handler struct {
	svc *service.Services
}

// New 初始化 Handler，由 main 调用，注入 Services 依赖
func New(svc *service.Services) *Handler {
	return &Handler{svc: svc}
}

// Register 注册所有路由。
// 路由分组规则：公共路由放顶层，业务路由统一挂载到 /api/v1 下。
func (h *Handler) Register(r *gin.Engine) {
	// 健康检查，不携带版本前缀，供 k8s/lb 探针直接访问
	r.GET("/health", h.Health)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/hello", h.Hello) // 打招呼示例接口：演示完整 handler→service→repository 链路

		// 采集器元信息
		v1.GET("/collectors", h.ListCollectors)

		// 任务 CRUD
		v1.GET("/tasks", h.ListTasks)
		v1.POST("/tasks", h.CreateTask)
		v1.GET("/tasks/:id", h.GetTask)
		v1.PUT("/tasks/:id", h.UpdateTask)
		v1.DELETE("/tasks/:id", h.DeleteTask)

		// 任务控制
		v1.POST("/tasks/:id/enable", h.EnableTask)
		v1.POST("/tasks/:id/disable", h.DisableTask)
		v1.POST("/tasks/:id/run", h.RunTaskNow)
	}
}
