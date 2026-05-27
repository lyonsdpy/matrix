# Matrix API

企业 IT 资产安全合规管理平台的后端服务。核心能力：

- **设备拓扑管理**：用 Neo4j 图数据库管理网络设备（交换机、路由器）及其互联关系，支持 BFS 拓扑遍历
- **员工组织同步**：从飞书通讯录实时同步员工/部门数据，存储于 PostgreSQL
- **安全合规检测**：对接 ACG 安全系统，检测设备软件黑名单违规，记录违规明细
- **GraphQL API**：设备、拓扑的完整 CRUD，DataLoader 解决 N+1 问题
- **REST API**：认证、员工查询、采集任务管理
- **事件驱动**：飞书 WebSocket 长连接接收设备变更、通讯录变更等事件；Kafka 用于消息解耦

---

## 技术栈

| 层次 | 选型 |
|------|------|
| HTTP 框架 | Gin |
| GraphQL | gqlgen v0.17 |
| 图数据库 | Neo4j 6（内存存根可降级） |
| 关系数据库 | PostgreSQL（sqlx + 自动迁移） |
| 消息队列 | Kafka（franz-go） |
| 飞书集成 | oapi-sdk-go v3（WebSocket 模式） |
| 认证 | JWT HS256 + AES-256-GCM 加密 Cookie |
| 日志 | zap + lumberjack |

---

## 目录结构

```
apps/api/
├── cmd/
│   ├── server/          # 主程序入口
│   └── seed/            # 数据初始化工具
├── configs/
│   └── config.yaml      # 服务配置（密码字段为 AES 密文）
├── domain/              # 核心领域模型（Device、User、Group、TemporalMeta…）
├── graph/               # GraphQL schema 及 gqlgen 生成代码
│   ├── *.graphqls       # Schema 定义
│   ├── device.resolvers.go  # 手写 resolver 实现
│   ├── resolver.go      # Resolver 依赖根
│   └── loader/          # DataLoader（批量加载，解决 N+1）
├── internal/
│   ├── collector/       # 采集器注册中心（HTTP / Shell）
│   ├── handler/         # HTTP handler 层（参数绑定 → Service → 响应）
│   ├── infra/
│   │   ├── aisacg/      # ACG 安全系统客户端
│   │   ├── desktop/     # Desktop 系统设备采集客户端
│   │   ├── kafka/       # Kafka Producer / Consumer 封装
│   │   ├── lark/        # 飞书 SDK 封装（事件分发、消息、联系人…）
│   │   ├── neo4j/       # Neo4j 连接及 Schema 初始化
│   │   └── postgres/    # PostgreSQL 连接及嵌入式迁移
│   ├── middleware/       # Gin 中间件（Auth、CORS、Logger、Recovery）
│   ├── model/           # HTTP 层专用请求/响应模型
│   ├── queue/           # 消息队列消费者处理逻辑
│   ├── repository/      # Repository 接口聚合及初始化
│   │   ├── neo4j_repo/  # Neo4j 图节点 CRUD（含内存存根）
│   │   └── pg_repo/     # PostgreSQL 数据访问（员工、设备、违规…）
│   ├── server/          # HTTP Server 封装（Init / Start / Stop）
│   └── service/         # 业务逻辑层
├── pkg/
│   ├── auth/            # JWT 生成/解析、密码哈希、用户上下文
│   ├── config/          # 配置加载（YAML + TryDecrypt）
│   ├── crypto/          # AES-256-GCM 加解密（配置密码 & Session Cookie）
│   ├── errno/           # 统一错误响应格式
│   └── log/             # 全局 Logger 初始化
└── examples/            # 独立可运行的示例（无需完整服务）
    ├── device/          # 设备图操作演示
    ├── group/           # 用户组操作演示
    └── gqlclient/       # GraphQL 客户端调用演示
```

---

## 分层规则

```
Handler → Service → Repository
```

- **Handler**：只做参数绑定、调用 Service、格式化响应，不含业务逻辑
- **Service**：业务规则、跨模块协调、错误语义包装
- **Repository**：只管数据存取，不知道任何业务规则
- **GraphQL Resolver**：解包 GraphQL 参数 → 调 Service → 转换为 GraphQL 类型

接口定义在**使用方**（Go 惯用法）：Service 声明自己需要的 repo 能力，不依赖具体实现。

---

## 快速开始

### 依赖

- Go 1.25+
- PostgreSQL 14+
- Neo4j 5+（可选，缺失时自动降级为内存存根）

### 配置

```yaml
# configs/config.yaml
postgres:
  host: localhost
  port: 5432
  user: matrix
  password: "your-password"   # 明文或 AES 密文均可
  db: matrix_db

neo4j:
  uri:      "bolt://localhost:7687"
  username: "neo4j"
  password: "your-password"

jwt:
  secret: "change-in-production"
  expiry_hours: 24
  secure_cookie: false   # 生产改为 true（HTTPS only）
```

**加密配置密码**（可选，推荐生产使用）：

```bash
# 生成加密后的密码，填入 config.yaml 对应字段
go run ./cmd/server -encrypt mypassword

# 生产环境须通过环境变量覆盖加密密钥（64 位十六进制字符串，32 字节）
export MATRIX_CRYPTO_KEY=<your-64-hex-chars>
```

### 启动

```bash
# 普通启动（自动执行数据库迁移）
go run ./cmd/server

# 重置 admin 账号密码
go run ./cmd/server -init-admin-password newpassword
```

服务默认监听 `:8080`。

---

## API 端点

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/login` | 账号密码登录，成功后写 HttpOnly cookie |
| POST | `/api/v1/auth/logout` | 清除 session cookie |

登录成功后 JWT 以 AES 加密写入 `matrix_session` cookie，GraphQL 端点通过此 cookie 鉴权。

### GraphQL

| 路径 | 说明 |
|------|------|
| `POST /graphql` | GraphQL 端点（需登录） |
| `GET /playground` | GraphQL Playground（开发调试） |

**主要 Query/Mutation**：

```graphql
# 查询设备列表（分页）
query { devices(first: 20) { nodes { id name mip } pageInfo { hasNextPage endCursor } } }

# 查询设备拓扑（BFS 2 跳）
query { deviceTopology(id: "xxx", depth: 2) { nodes { id props } edges { from to type } } }

# 创建设备并建立连接
mutation { createDevice(name: "SW-01", deviceType: "switch", mip: "10.0.0.1", connectTo: ["id1"]) { id } }
```

### REST

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查（k8s 探针） |
| GET | `/api/v1/employees/:id` | 查询员工（PG + 图 join） |
| GET | `/api/v1/collectors` | 列出已注册采集器类型 |
| GET | `/api/v1/tasks` | 任务列表 |
| POST | `/api/v1/tasks` | 创建采集任务 |
| PUT | `/api/v1/tasks/:id` | 更新任务配置 |
| DELETE | `/api/v1/tasks/:id` | 删除任务 |
| POST | `/api/v1/tasks/:id/enable` | 启用任务 |
| POST | `/api/v1/tasks/:id/disable` | 禁用任务 |
| POST | `/api/v1/tasks/:id/run` | 立即触发一次采集 |

---

## 数据库迁移

迁移文件嵌入到二进制中，服务启动时**自动执行**，无需手动操作：

```
internal/infra/postgres/migrations/
├── 000001_create_departments.*
├── 000002_create_employees.*
├── 000003_create_employee_devices.*
├── 000004_create_online_sessions.*
├── 000005_create_violation_records.*
├── 000006_create_software_blacklist.*
├── 000007_create_acg_whitelist.*
├── 000008_create_user_bindings.*
├── 000009_create_bot_interactions.*
└── 000010_create_users.*
```

---

## 新增业务模块

参考 `internal/service/README.md`，三步扩展：

1. `internal/repository/pg_repo/xxx.go` — 定义 Repository 接口 + 实现
2. `internal/service/xxx.go` — 定义 Service 接口 + 实现，声明所需的 repo 接口
3. `internal/handler/xxx.go` — 添加 Handler 方法，在 `handler.go#Register` 注册路由

GraphQL 扩展：修改 `graph/*.graphqls` → 运行 `go generate ./graph/` → 实现生成的 resolver 方法。

---

## 开发说明

### Neo4j 降级

`neo4j.uri` 留空时，服务自动使用**内存存根**，无需启动 Neo4j。存根实现与生产实现共享同一接口，适合本地开发和单元测试。

### 运行示例

```bash
# 设备图操作（内存存根，无需任何依赖）
go run ./examples/device/...

# 用户组操作
go run ./examples/group/...
```

### 生成 GraphQL 代码

```bash
go generate ./graph/
```
