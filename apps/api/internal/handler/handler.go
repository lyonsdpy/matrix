package handler

import (
	"context"
	"net/http"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"matrix/api/graph"
	"matrix/api/graph/loader"
	"matrix/api/internal/repository"
	"matrix/api/internal/service"
)

// Handler 聚合所有 HTTP 处理方法，持有 Services 引用。
// 职责：参数绑定、调用 Service、返回 HTTP 响应，不含业务逻辑。
type Handler struct {
	svc        *service.Services
	repos      *repository.Repositories
	gqlHandler http.Handler // gqlgen 生成的 GraphQL 执行引擎，进程级单例
}

// New 初始化 Handler，由 main 调用，注入 Services 和 Repositories 依赖
func New(svc *service.Services, repos *repository.Repositories) *Handler {
	resolver := graph.NewResolver(repos.Device)
	es := graph.NewExecutableSchema(graph.Config{Resolvers: resolver})

	return &Handler{
		svc:        svc,
		repos:      repos,
		gqlHandler: gqlhandler.NewDefaultServer(es),
	}
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

	// GraphQL 端点
	// DataLoader 中间件：每个请求创建独立的 Loaders 实例注入 context，
	// 保证 DataLoader 的批次不跨请求
	gql := r.Group("/graphql")
	gql.Use(h.dataLoaderMiddleware())
	gql.POST("", gin.WrapH(h.gqlHandler))
	gql.GET("", gin.WrapH(h.gqlHandler))

	// GraphQL Playground：开发环境调试，生产可通过配置禁用
	r.GET("/playground", gin.WrapH(playground.Handler("GraphQL Playground", "/graphql")))
}

// dataLoaderMiddleware 为每个请求创建新的 DataLoader 集合并注入 context。
// per-request 创建是关键：DataLoader 的批量窗口和缓存都是请求隔离的。
func (h *Handler) dataLoaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		loaders := loader.New(h.repos.Device)
		ctx := context.WithValue(c.Request.Context(), loader.Key, loaders)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
