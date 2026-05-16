# migrations/

数据库迁移文件。使用 SQL 文件管理 schema 变更历史，保证每次迁移可追溯、可回滚。

## 文件命名规范

```
migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── 000002_create_devices.up.sql
├── 000002_create_devices.down.sql
└── 000003_add_device_status.up.sql
    000003_add_device_status.down.sql
```

格式：`{序号}_{描述}.{up|down}.sql`

- `up.sql`：正向迁移（应用变更）
- `down.sql`：回滚迁移（撤销变更）
- 序号 6 位补零，全局递增

## 推荐工具

使用 **[golang-migrate](https://github.com/golang-migrate/migrate)** 管理：

```bash
# 应用所有未执行的迁移
migrate -path ./migrations -database "postgres://..." up

# 回滚最后一次迁移
migrate -path ./migrations -database "postgres://..." down 1
```

也可在 `Makefile` 中封装：

```makefile
migrate-up:
    migrate -path apps/api/migrations -database $(DATABASE_URL) up

migrate-down:
    migrate -path apps/api/migrations -database $(DATABASE_URL) down 1
```

## 约定

- 已合并的迁移文件不可修改，需变更则新建迁移
- 每次迁移只做一件事（不把多个表改动塞进一个迁移）
- `down.sql` 必须完整实现回滚，不可留空
