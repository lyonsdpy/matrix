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
	svc            *service.Services
	deviceRepo     repository.DeviceGraphRepo // 仅供 DataLoader middleware 使用，不做业务调用
	jwtSecret      string
	internalSecret string // Next.js 服务端调内部接口时的共享密钥
	gqlHandler     http.Handler
}

// New 初始化 Handler，由 main 调用，注入 Services 和依赖配置。
func New(svc *service.Services, deviceRepo repository.DeviceGraphRepo, jwtCfg config.JWT, larkCfg config.Lark) *Handler {
	resolver := graph.NewResolver(graph.Deps{DeviceSvc: svc.Device})
	es := graph.NewExecutableSchema(graph.Config{Resolvers: resolver})

	return &Handler{
		svc:            svc,
		deviceRepo:     deviceRepo,
		jwtSecret:      jwtCfg.Secret,
		internalSecret: larkCfg.InternalSecret,
		gqlHandler:     newGQLServer(es),
	}
}

// Register 注册所有路由。
func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", h.Health)

	// 认证路由：账号密码登录，Next.js 服务端调用后写 cookie
	authGroup := r.Group("/api/v1/auth")
	{
		authGroup.POST("/login", h.Login)
	}

	// ACG 终端合规检查：飞书认证之前的前置门户调用，调用方尚未持有 JWT，
	// 因此与登录接口一样属公开端点，不挂 Auth 中间件。
	acg := r.Group("/api/v1/acg")
	{
		acg.POST("/edr-check", h.CheckEDR)
	}

	v1 := r.Group("/api/v1")
	v1.Use(middleware.Auth(h.jwtSecret))
	{
		// ── 仅 Auth 的接口（演示/基础设施，暂未划入权限模块） ──
		v1.GET("/hello", h.Hello)
		v1.GET("/collectors", h.ListCollectors)
		v1.GET("/tasks", h.ListTasks)
		v1.POST("/tasks", h.CreateTask)
		v1.GET("/tasks/:id", h.GetTask)
		v1.PUT("/tasks/:id", h.UpdateTask)
		v1.DELETE("/tasks/:id", h.DeleteTask)
		v1.POST("/tasks/:id/enable", h.EnableTask)
		v1.POST("/tasks/:id/disable", h.DisableTask)
		v1.POST("/tasks/:id/run", h.RunTaskNow)
		v1.GET("/employees/:id", h.GetEmployee)
		// 当前登录者的权限码集合：登录后必查，给前端按钮/菜单显隐用
		v1.GET("/me/permissions", h.MyPermissions)

		perm := h.svc.Permission // 中间件依赖注入

		// ── 通讯录（飞书用户/部门） ──────────────────────────────
		v1.GET("/users", middleware.RequirePermission(perm, "contact:read"), h.ListUsers)
		v1.GET("/users/:id", middleware.RequirePermission(perm, "contact:read"), h.GetUserDetail)
		// 系统本地账号（PG users 表）相关：路径前缀 /auth-users 与通讯录 /users 解耦，避免 :id 语义混淆
		v1.GET("/auth-users/by-lark/:openId", middleware.RequirePermission(perm, "user:read"), h.GetAuthUserByLark)
		v1.GET("/auth-users/:id/roles", middleware.RequirePermission(perm, "user:read"), h.GetAuthUserRoles)
		v1.PUT("/auth-users/:id/roles", middleware.RequirePermission(perm, "user:role:assign"), h.SetAuthUserRoles)
		v1.GET("/departments/:id", middleware.RequirePermission(perm, "contact:read"), h.GetDepartmentDetail)
		v1.GET("/contacts/tree", middleware.RequirePermission(perm, "contact:read"), h.ListDepartmentChildren)
		v1.GET("/contacts/search", middleware.RequirePermission(perm, "contact:read"), h.SearchContacts)
		v1.POST("/contacts/sync/start", middleware.RequirePermission(perm, "contact:sync"), h.StartContactSync)
		v1.GET("/contacts/sync/progress", middleware.RequirePermission(perm, "contact:read"), h.GetContactSyncProgress)

		// ── 终端管理 ─────────────────────────────────────────────
		v1.GET("/endpoints", middleware.RequirePermission(perm, "endpoint:read"), h.ListEndpoints)
		v1.GET("/endpoints/:id", middleware.RequirePermission(perm, "endpoint:read"), h.GetEndpoint)
		v1.POST("/endpoints/sync/start", middleware.RequirePermission(perm, "endpoint:sync"), h.StartEndpointSync)
		v1.GET("/endpoints/sync/progress", middleware.RequirePermission(perm, "endpoint:read"), h.GetEndpointSyncProgress)

		// ── 权限/角色管理 ────────────────────────────────────────
		v1.GET("/permissions", middleware.RequirePermission(perm, "role:read"), h.ListPermissions)
		v1.GET("/roles", middleware.RequirePermission(perm, "role:read"), h.ListRoles)
		v1.GET("/roles/:id", middleware.RequirePermission(perm, "role:read"), h.GetRole)
		v1.POST("/roles", middleware.RequirePermission(perm, "role:write"), h.CreateRole)
		v1.PUT("/roles/:id", middleware.RequirePermission(perm, "role:write"), h.UpdateRole)
		v1.DELETE("/roles/:id", middleware.RequirePermission(perm, "role:delete"), h.DeleteRole)
		v1.PUT("/roles/:id/permissions", middleware.RequirePermission(perm, "role:assign"), h.SetRolePermissions)
		v1.GET("/roles/:id/users", middleware.RequirePermission(perm, "role:read"), h.ListRoleUsers)
		v1.POST("/roles/:id/users", middleware.RequirePermission(perm, "role:assign"), h.AddRoleUsers)
		v1.DELETE("/roles/:id/users/:userId", middleware.RequirePermission(perm, "role:assign"), h.RemoveRoleUser)
	}

	// 内部接口：仅供 Next.js 服务端调用，通过 X-Internal-Secret 鉴权，不走 JWT 中间件
	internal := r.Group("/internal")
	{
		internal.POST("/auth/lark/exchange", h.LarkExchange)
	}

	// GraphQL 端点：JWT 鉴权 + DataLoader 批量加载
	gql := r.Group("/graphql")
	gql.Use(middleware.Auth(h.jwtSecret))
	gql.Use(h.dataLoaderMiddleware())
	gql.POST("", gin.WrapH(h.gqlHandler))
	gql.GET("", gin.WrapH(h.gqlHandler))

	r.GET("/playground", gin.WrapH(playground.Handler("GraphQL Playground", "/graphql")))
}

func (h *Handler) dataLoaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		loaders := loader.NewLoaders(h.deviceRepo)
		ctx := context.WithValue(c.Request.Context(), loader.Key, loaders)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

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
