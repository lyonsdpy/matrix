# matrix

## 项目说明

这是一个monorepo，功能是一个带CMDB的网络管理系统，当前的项目框架如下

```text
matrix/
├── apps/                         # 应用目录
│   ├── web/                      # Next.js 前端项目
│   │   ├── app/                  # App Router 页面与路由
│   │   ├── components/           # React 组件（基于 MUI）
│   │   ├── theme/                # MUI 主题配置（ThemeProvider、调色板、排版）
│   │   ├── lib/                  # 纯工具函数（格式化、日期、校验等）
│   │   ├── services/             # API 请求封装（fetch wrappers）
│   │   ├── hooks/                # React 自定义 hooks
│   │   ├── types/                # TypeScript 类型定义
│   │   ├── styles/               # 全局样式
│   │   ├── public/               # 静态资源
│   │   ├── package.json
│   │   ├── tsconfig.json
│   │   └── next.config.ts
│   │
│   └── api/                      # Go Gin 后端项目
│       ├── cmd/                  # 程序入口
│       │   └── server/
│       │       └── main.go
│       ├── internal/             # 内部业务代码
│       │   ├── handler/          # HTTP Handler（请求/响应处理）
│       │   ├── service/          # 业务逻辑层
│       │   ├── repository/       # 数据访问层
│       │   ├── middleware/       # Gin 中间件（认证、限流、日志等）
│       │   ├── authz/            # 权限模型与访问控制（RBAC）
│       │   └── model/            # 数据模型（ORM 结构体）
│       ├── pkg/                  # 可复用公共库
│       ├── configs/              # 配置文件
│       ├── migrations/           # 数据库 migration
│       ├── go.mod
│       └── go.sum
│
├── packages/                     # Monorepo 共享包
│   └── types/                    # 前后端共享类型（可由 API schema 生成）
│
├── deploy/                       # 部署相关
│   ├── docker/                   # Dockerfile
│   ├── compose/                  # docker-compose
│   ├── k8s/                      # Kubernetes YAML
│   └── scripts/                  # 部署/构建脚本
│
├── docs/                         # 项目文档
│   ├── architecture/             # 架构设计
│   ├── api/                      # API 文档
│   ├── deploy/                   # 部署文档
│   └── authz/                    # 权限模型设计（RBAC）
│
├── .env.example                  # 环境变量模板
├── .gitignore
├── README.md
└── Makefile
```

