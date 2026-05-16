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
