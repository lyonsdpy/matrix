# docs/deploy/

部署文档。

## 常见文件

```
deploy/
├── setup.md        # 首次部署步骤（环境准备、依赖安装）
└── env-vars.md     # 所有环境变量说明
```

## env-vars.md 格式约定

| 变量名 | 必填 | 默认值 | 说明 |
|---|---|---|---|
| `DATABASE_URL` | 是 | — | PostgreSQL 连接串 |
| `REDIS_ADDR` | 否 | `localhost:6379` | Redis 地址 |
| `JWT_SECRET` | 是 | — | JWT 签名密钥，至少 32 字符 |
| `APP_ENV` | 否 | `development` | `development` / `production` |
