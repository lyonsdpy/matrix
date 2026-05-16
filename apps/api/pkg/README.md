# pkg/

可复用公共库。不包含业务逻辑，可被 `internal/` 的任意包引用，理论上也可提取为独立模块。

## 常见子目录

```
pkg/
├── response/       # 统一 API 响应格式
│   └── response.go
├── logger/         # 日志初始化与封装（基于 zap 或 slog）
│   └── logger.go
├── config/         # 配置加载（基于 viper）
│   └── config.go
├── errors/         # 自定义错误类型与错误码
│   └── errors.go
└── pagination/     # 通用分页参数解析
    └── pagination.go
```

## 示例：response/response.go

```go
type Response[T any] struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    T      `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
    c.JSON(200, Response[any]{Code: 0, Message: "ok", Data: data})
}

func BadRequest(c *gin.Context, err error) {
    c.JSON(400, Response[any]{Code: 400, Message: err.Error()})
}
```

## 约定

- `pkg/` 中的包不引用 `internal/` 中的任何内容（单向依赖）
- 每个子目录是一个独立的 Go 包，职责单一
- 不放业务相关的代码（如果某段代码依赖特定业务模型，它属于 `internal/`）
