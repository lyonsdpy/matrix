# docs/api/

API 接口文档。

## 常见文件

```
api/
├── openapi.yaml        # OpenAPI 3.0 规范文件（接口契约）
└── examples/           # 请求/响应示例
    └── devices.json
```

## 工具推荐

- **预览**：`swagger-ui` 或 VS Code 插件 `OpenAPI (Swagger) Editor`
- **类型生成**：`openapi-typescript` 从 `openapi.yaml` 生成 `packages/types` 中的 TS 类型
- **Mock**：`prism` 根据规范启动 mock server，前端可在后端未完成时并行开发

```bash
# 生成 TypeScript 类型
npx openapi-typescript docs/api/openapi.yaml -o packages/types/src/generated.ts

# 启动 mock server
npx @stoplight/prism-cli mock docs/api/openapi.yaml
```

## 约定

- 接口变更先更新 `openapi.yaml`，再同步实现（接口契约优先）
- `openapi.yaml` 提交到 git，作为前后端协作的唯一真实来源

---

## 现有接口列表

### 健康检查

```
GET /health
```

响应：
```json
{"status": "ok"}
```

---

### Hello World（服务链演示）

```
GET /api/v1/hello?name={姓名}
```

| 参数   | 位置  | 类型   | 必填 | 说明   |
|--------|-------|--------|------|--------|
| `name` | query | string | 是   | 用户姓名 |

**成功响应 200：**
```json
{
  "message": "Hello, Matrix! 欢迎使用 Matrix API。"
}
```

**参数缺失响应 400：**
```json
{
  "code": 400,
  "error": "Key: 'HelloRequest.Name' Error:Field validation for 'Name' failed on the 'required' tag",
  "msg": "参数错误"
}
```

**服务链路说明：**

```
HTTP 请求
    ↓
Handler（apps/api/internal/handler/hello.go）
    绑定 query 参数、调用 Service、返回 JSON
    ↓
Service（apps/api/internal/service/hello.go）
    封装业务规则（参数校验、国际化、缓存等扩展点）
    ↓
Repository（apps/api/internal/repository/hello.go）
    数据访问层（当前为内存实现，可替换为 DB 查询）
    ↓
返回数据，逐层回传
```
