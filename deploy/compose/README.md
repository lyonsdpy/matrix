# deploy/compose/

Docker Compose 配置。用于本地开发环境和集成测试。

## 常见文件

```
compose/
├── docker-compose.yml          # 基础服务（数据库、Redis 等基础设施）
└── docker-compose.override.yml # 本地开发覆写（热重载、端口映射等）
```

## 启动开发环境

```bash
# 启动所有基础设施（数据库、Redis）
docker compose -f deploy/compose/docker-compose.yml up -d

# 停止
docker compose -f deploy/compose/docker-compose.yml down
```

推荐在根目录 `Makefile` 中封装：

```makefile
dev-infra-up:
    docker compose -f deploy/compose/docker-compose.yml up -d

dev-infra-down:
    docker compose -f deploy/compose/docker-compose.yml down
```

## 约定

- Compose 文件只用于开发/测试，生产环境使用 k8s
- 数据库 volume 挂载到本地目录，重启不丢数据
- 端口统一在此文件中定义，避免与系统服务冲突
