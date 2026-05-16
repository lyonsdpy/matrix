# docs/authz/

权限模型设计文档。说明系统的 RBAC 设计、角色定义和权限矩阵。

## 常见文件

```
authz/
├── rbac-model.md       # RBAC 模型说明（角色、资源、操作的定义）
└── permissions.md      # 权限矩阵（哪个角色能对哪个资源做什么操作）
```

## rbac-model.md 内容参考

```
核心概念：
- Subject（主体）：发起操作的用户
- Role（角色）：admin / operator / viewer
- Resource（资源）：device / network / cmdb-ci / user
- Action（操作）：read / write / delete

关系：
User ──拥有──▶ Role ──授权──▶ (Resource, Action)
```

## permissions.md 内容参考（权限矩阵）

| 资源 \ 角色 | admin | operator | viewer |
|---|---|---|---|
| device:read | ✓ | ✓ | ✓ |
| device:write | ✓ | ✓ | — |
| device:delete | ✓ | — | — |
| user:manage | ✓ | — | — |

## 约定

- 权限矩阵是系统行为的规格说明，`internal/authz/` 的实现必须与此保持一致
- 新增资源或操作时，先更新此文档，再修改代码
