# packages/types/

前后端共享的 TypeScript 类型定义。此包是唯一的"合同层"——后端 API 变更时，先更新这里，前端编译器会自动提示所有受影响的调用点。

## 常见文件

```
types/
├── src/
│   ├── models.ts       # 核心领域模型（Device、Network、CiItem、User）
│   ├── api.ts          # API 请求/响应通用结构
│   └── index.ts        # 统一导出
├── package.json
└── tsconfig.json
```

## 示例内容

```ts
// models.ts
export interface Device {
  id: string
  hostname: string
  ip: string
  status: DeviceStatus
  createdAt: string
  updatedAt: string
}

export type DeviceStatus = 'online' | 'offline' | 'unknown'

// api.ts
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface PaginatedResult<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}
```

## 如何在 web 中引用

```ts
import type { Device, ApiResponse } from '@matrix/types'
```

## 约定

- 只放纯类型，不含任何运行时代码（无函数、无常量）
- 类型变更需同时更新后端接口和前端调用，视为破坏性变更
- 可通过 OpenAPI schema（`docs/api/openapi.yaml`）自动生成，减少手动维护
