# internal/repository/

数据访问层。封装所有数据库操作，向上只暴露接口，屏蔽 ORM 实现细节。

## 常见文件

```
repository/
├── device_repo.go      # 设备表 CRUD（含接口定义）
├── network_repo.go     # 网络表 CRUD
├── cmdb_repo.go        # CMDB CI 表 CRUD
└── user_repo.go        # 用户表
```

## 文件结构约定

```go
// 接口定义
type DeviceRepository interface {
    FindAll(ctx context.Context, filter DeviceFilter) ([]*model.Device, int64, error)
    FindByID(ctx context.Context, id string) (*model.Device, error)
    Create(ctx context.Context, device *model.Device) error
    Update(ctx context.Context, device *model.Device) error
    Delete(ctx context.Context, id string) error
}

// GORM 实现
type deviceRepository struct {
    db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) DeviceRepository {
    return &deviceRepository{db: db}
}
```

## 约定

- 只做数据读写，不包含业务逻辑（如条件判断、状态机流转）
- 接口定义在此包内，service 层依赖接口而非实现（便于单元测试 mock）
- 分页查询统一返回 `(items, total, error)`
- 软删除通过 GORM 的 `DeletedAt` 字段实现，不直接 DELETE
