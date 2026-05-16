# internal/handler/

HTTP Handler 层。负责解析请求、调用 service、返回响应。不包含任何业务逻辑。

## 常见文件

```
handler/
├── device_handler.go       # 设备相关接口
├── network_handler.go      # 网络相关接口
├── cmdb_handler.go         # CMDB CI 接口
├── auth_handler.go         # 登录、登出、token 刷新
└── router.go               # 路由注册（将 handler 挂载到 gin.Engine）
```

## Handler 职责

```
请求进入
 ├── 绑定并校验请求参数（gin.ShouldBindJSON / ShouldBindQuery）
 ├── 提取上下文信息（用户 ID、Trace ID）
 ├── 调用 service 层
 └── 返回统一格式响应（pkg/response）
```

## 示例结构

```go
type DeviceHandler struct {
    svc service.DeviceService
}

func (h *DeviceHandler) List(c *gin.Context) {
    var req dto.ListDevicesRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        response.BadRequest(c, err)
        return
    }
    result, err := h.svc.List(c.Request.Context(), req)
    if err != nil {
        response.InternalError(c, err)
        return
    }
    response.OK(c, result)
}
```

## 约定

- Handler 不直接操作数据库，不包含 SQL 或 ORM 调用
- 参数校验失败立即返回，不进入 service
- 每个 handler 文件对应一个资源，方法名对应 HTTP 动作（List、Get、Create、Update、Delete）
