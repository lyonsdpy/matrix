# cmd/server/

程序入口。职责仅限于启动：加载配置、初始化依赖、组装路由、启动 HTTP Server。

## 常见文件

```
cmd/server/
└── main.go     # 程序入口
```

## main.go 职责

```
main()
 ├── 加载配置（configs/）
 ├── 初始化数据库连接
 ├── 初始化 Redis（如有）
 ├── 依赖注入：repo → service → handler
 ├── 注册路由与中间件
 └── 启动 gin.Engine
```

## 约定

- `main.go` 只做组装，不写业务逻辑
- 保持简短（通常 < 100 行），复杂初始化逻辑抽到 `internal/` 的专用函数中
- 优雅关闭：监听 `SIGTERM`/`SIGINT`，等待在途请求完成后退出
