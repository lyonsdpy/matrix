# internal/model/

数据模型。定义与数据库表对应的 Go 结构体（ORM Model）及相关 DTO（数据传输对象）。

## 常见文件

```
model/
├── device.go       # 设备模型
├── network.go      # 网络模型
├── cmdb.go         # CMDB CI 模型
├── user.go         # 用户模型
└── dto/            # 请求/响应 DTO（与 model 分离）
    ├── device_dto.go
    └── network_dto.go
```

## 文件结构示例

```go
// device.go — 数据库模型
type Device struct {
    gorm.Model             // ID, CreatedAt, UpdatedAt, DeletedAt
    Hostname  string       `gorm:"uniqueIndex;not null"`
    IP        string       `gorm:"not null"`
    Status    DeviceStatus `gorm:"default:unknown"`
    SiteID    uint
    Site      Site         `gorm:"foreignKey:SiteID"`
}

// dto/device_dto.go — 请求/响应对象
type CreateDeviceRequest struct {
    Hostname string `json:"hostname" binding:"required"`
    IP       string `json:"ip"       binding:"required,ip"`
}

type DeviceResponse struct {
    ID       uint   `json:"id"`
    Hostname string `json:"hostname"`
    IP       string `json:"ip"`
    Status   string `json:"status"`
}
```

## 约定

- ORM Model（`model/`）和传输对象（`dto/`）分离，不直接将数据库模型暴露给 API
- Model 不引用 handler 或 service 包，保持最底层，无上游依赖
- 枚举字段（如 `DeviceStatus`）定义在同文件内，使用 `string` 类型常量
