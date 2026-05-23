package handler

import (
	"context"
	"net/http"

	"matrix/api/graph"
	"matrix/api/graph/loader"
	"matrix/api/internal/middleware"
	"matrix/api/internal/repository"
	"matrix/api/internal/service"
	"matrix/api/pkg/config"

	"github.com/99designs/gqlgen/graphql"
	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/vektah/gqlparser/v2/ast"
)

// Handler 聚合所有 HTTP 处理方法，持有 Services 引用。
// 职责：参数绑定、调用 Service、返回 HTTP 响应，不含业务逻辑。
type Handler struct {
	svc          *service.Services
	repos        *repository.Repositories
	jwtSecret    string
	secureCookie bool
	gqlHandler   http.Handler // gqlgen 生成的 GraphQL 执行引擎，进程级单例
}

// New 初始化 Handler，由 main 调用，注入 Services 和 Repositories 依赖
func New(svc *service.Services, repos *repository.Repositories, jwtCfg config.JWT) *Handler {
	// Resolver 依赖 Service 而不是 Repository——GraphQL 层不直接碰数据库
	resolver := graph.NewResolver(svc.Device)
	es := graph.NewExecutableSchema(graph.Config{Resolvers: resolver})

	return &Handler{
		svc:          svc,
		repos:        repos,
		jwtSecret:    jwtCfg.Secret,
		secureCookie: jwtCfg.SecureCookie,
		gqlHandler:   newGQLServer(es),
	}
}

// Register 注册所有路由。
// 路由分组规则：公共路由放顶层，业务路由统一挂载到 /api/v1 下。
func (h *Handler) Register(r *gin.Engine) {
	// 健康检查，不携带版本前缀，供 k8s/lb 探针直接访问
	r.GET("/health", h.Health)

	// 认证路由：不需要 JWT，开放访问
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/logout", h.Logout)
	}

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

		// 员工查询（跨源：PG 同步数据 + 图 User 节点，service 层 join）
		v1.GET("/employees/:id", h.GetEmployee)
	}

	// GraphQL 端点：JWT 鉴权 + DataLoader 批量加载
	gql := r.Group("/graphql")
	gql.Use(middleware.Auth(h.jwtSecret))
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
		loaders := loader.NewLoaders(h.repos.Graph.Device)
		ctx := context.WithValue(c.Request.Context(), loader.Key, loaders)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// newGQLServer 用显式配置替代已废弃的 NewDefaultServer。
// 不注册 WebSocket transport——当前路由只有 POST/GET，SSE/WS 均未启用。
func newGQLServer(es graphql.ExecutableSchema) *gqlhandler.Server {
	srv := gqlhandler.New(es)
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})
	return srv
}
