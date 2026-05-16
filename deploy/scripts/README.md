# deploy/scripts/

部署与构建脚本。复杂的多步操作封装为脚本，在 `Makefile` 中调用。

## 常见文件

```
scripts/
├── build.sh        # 构建所有服务镜像并推送到镜像仓库
├── deploy.sh       # 部署到指定环境（需传入 ENV 参数）
└── migrate.sh      # 执行数据库迁移（需传入 DATABASE_URL）
```

## 使用示例

```bash
# 构建并推送镜像（通常由 CI 执行）
bash deploy/scripts/build.sh v1.2.0

# 部署到 staging
ENV=staging bash deploy/scripts/deploy.sh

# 执行数据库迁移
DATABASE_URL=postgres://... bash deploy/scripts/migrate.sh
```

## 约定

- 脚本第一行写 `set -euo pipefail`，任意步骤失败立即终止
- 脚本接受环境变量而非位置参数（更易在 CI 中配置）
- 脚本需幂等：重复执行不产生副作用
