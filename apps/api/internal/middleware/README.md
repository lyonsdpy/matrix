# internal/middleware/

Gin 中间件。处理横切关注点：认证验证、日志、跨域、限流等。

## 常见文件

```
middleware/
├── auth.go         # JWT 校验，将用户信息注入 gin.Context
├── cors.go         # CORS 配置
├── logger.go       # 请求日志（请求方法、路径、耗时、状态码）
├── recovery.go     # panic 恢复，返回 500 而非进程崩溃
└── rate_limiter.go # 接口限流
```

## 中间件示意

```go
// auth.go
func Auth(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        claims, err := parseJWT(token, secret)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"message": "unauthorized"})
            return
        }
        c.Set("userID", claims.UserID)
        c.Set("roles", claims.Roles)
        c.Next()
    }
}
```

## 约定

- 中间件只读取/注入 `gin.Context`，不调用 service 或 repository（auth 中间件除外）
- 权限判断（某用户是否有权访问某资源）不在此层做，交给 `authz/` 处理
- 中间件在 `handler/router.go` 中按需挂载到路由组，不全局强制应用
