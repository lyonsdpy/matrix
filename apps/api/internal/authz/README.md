# internal/authz/

权限模型与访问控制。实现 RBAC（基于角色的访问控制），判断特定用户是否有权执行某操作。

## 常见文件

```
authz/
├── enforcer.go     # 权限检查入口（基于 Casbin 或自研）
├── policy.go       # 策略加载与更新（从数据库同步策略）
├── roles.go        # 角色定义与角色-权限映射
└── model.conf      # Casbin 模型配置文件（如使用 Casbin）
```

## 职责边界

```
middleware/auth.go          → 验证 token 合法性（你是谁）
internal/authz/enforcer.go  → 检查操作权限（你能做什么）
```

## 核心接口

```go
type Enforcer interface {
    // 检查 userID 是否可以对 resource 执行 action
    // action: "read" | "write" | "delete"
    Enforce(ctx context.Context, userID, resource, action string) (bool, error)

    // 给用户分配角色
    AssignRole(ctx context.Context, userID, role string) error

    // 获取用户所有权限
    GetPermissions(ctx context.Context, userID string) ([]Permission, error)
}
```

## 约定

- 权限策略持久化到数据库，服务启动时加载，变更时热更新
- Handler 层通过调用 `enforcer.Enforce()` 做权限检查，或封装为 gin 中间件按路由组挂载
- 角色定义集中在 `roles.go`，不散落在业务代码中
