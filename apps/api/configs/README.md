# configs/

配置文件目录。存放应用配置，按环境区分。

## 常见文件

```
configs/
├── config.yaml         # 默认配置（本地开发）
├── config.prod.yaml    # 生产环境覆写（仅差异项）
└── config.example.yaml # 配置模板（提交到 git，含所有字段说明）
```

## 配置结构示例

```yaml
server:
  port: 8080
  mode: debug          # debug | release

database:
  host: localhost
  port: 5432
  name: matrix
  user: postgres
  password: ""         # 生产环境从环境变量注入

redis:
  addr: localhost:6379

jwt:
  secret: ""           # 必须从环境变量注入
  expiry: 24h
```

## 约定

- 敏感字段（密码、密钥）在配置文件中留空，通过环境变量覆写（viper 支持 `MATRIX_DATABASE_PASSWORD` → `database.password`）
- `config.yaml` 可提交到 git（本地开发默认值），`config.prod.yaml` 不提交（gitignore）
- `config.example.yaml` 必须保持更新，新增配置项时同步添加注释说明
