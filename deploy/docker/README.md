# deploy/docker/

各服务的 Dockerfile。

## 常见文件

```
docker/
├── Dockerfile.web      # Next.js 前端（多阶段构建）
└── Dockerfile.api      # Go 后端（多阶段构建）
```

## 构建示例

```bash
# 构建后端镜像
docker build -f deploy/docker/Dockerfile.api -t matrix-api:latest .

# 构建前端镜像
docker build -f deploy/docker/Dockerfile.web -t matrix-web:latest .
```

## 约定

- 使用多阶段构建（builder + runtime），最终镜像不含编译工具链
- Go 镜像基于 `gcr.io/distroless/static` 或 `alpine`，减小体积
- Next.js 镜像使用 `node:alpine` + `output: standalone` 模式
- 不在 Dockerfile 中硬编码任何密钥或环境变量，运行时通过 `-e` 或 secret 注入
