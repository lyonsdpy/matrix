# internal/service/

业务逻辑层。编排多个 repository 调用，实现具体业务规则。

## 常见文件

```
service/
├── device_service.go       # 设备业务逻辑（含接口定义）
├── network_service.go      # 网络业务逻辑
├── cmdb_service.go         # CMDB CI 管理
└── auth_service.go         # 认证：密码校验、token 签发
```

## 文件结构约定

每个文件同时包含接口和实现，便于 mock 测试：

```go
// 接口定义
type DeviceService interface {
    List(ctx context.Context, req dto.ListDevicesRequest) (*dto.PaginatedResult[model.Device], error)
    GetByID(ctx context.Context, id string) (*model.Device, error)
    Create(ctx context.Context, req dto.CreateDeviceRequest) (*model.Device, error)
}

// 实现结构体
type deviceService struct {
    repo repository.DeviceRepository
}

func NewDeviceService(repo repository.DeviceRepository) DeviceService {
    return &deviceService{repo: repo}
}
```

## 约定

- 不直接调用数据库，必须通过 repository 接口
- 不处理 HTTP 细节（不引用 `gin.Context`，参数通过 context.Context + DTO 传递）
- 跨资源的事务性操作在 service 层协调，不在 handler 层
- 业务错误返回自定义错误类型（如 `ErrDeviceNotFound`），由 handler 层映射为 HTTP 状态码
