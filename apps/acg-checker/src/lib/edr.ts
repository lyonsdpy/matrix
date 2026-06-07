export type EdrResult = 'pass' | 'fail'

export interface EdrCheckOptions {
  apiBaseUrl?: string
  timeoutMs?: number
}

// 后端接口路径与 apps/api 中 handler.go 注册保持一致
const ENDPOINT_PATH = '/api/v1/acg/edr-check'
const DEFAULT_TIMEOUT_MS = 5000

// 后端响应体结构，与 service.EDRCheckResult 字段对齐
interface EDRCheckResponse {
  pass: boolean
  reason?: string
}

function resolveApiBaseUrl(opts: EdrCheckOptions): string {
  // 显式入参优先，便于测试与本地调试覆盖；其次取构建期注入的 VITE_API_BASE_URL
  const fromOpts = opts.apiBaseUrl?.trim()
  if (fromOpts) return fromOpts.replace(/\/+$/, '')
  const fromEnv = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim()
  if (fromEnv) return fromEnv.replace(/\/+$/, '')
  // 兜底：相对路径，要求 acg-checker 与后端同源（生产形态一般另用 VITE_API_BASE_URL）
  return ''
}

// checkEdr 让后端探测当前客户端 IP 的 OfficeScan 端口，按 503 + Server: OfficeScan Client
// 严格判定。浏览器侧不再做 no-cors fetch，避免无法读 status/header 的限制。
export async function checkEdr(opts: EdrCheckOptions = {}): Promise<EdrResult> {
  // 本地开发 mock 短路：留作离线/无网环境演示用
  const mock = import.meta.env.VITE_EDR_MOCK
  if (mock === 'pass' || mock === 'fail') {
    if (import.meta.env.DEV) {
      console.info('[edr] mocked result', mock)
    }
    return mock
  }

  const url = `${resolveApiBaseUrl(opts)}${ENDPOINT_PATH}`
  const timeoutMs = opts.timeoutMs ?? DEFAULT_TIMEOUT_MS
  const startedAt = performance.now()

  const ac = new AbortController()
  const timer = setTimeout(() => ac.abort(), timeoutMs)

  try {
    const resp = await fetch(url, {
      method: 'POST',
      cache: 'no-store',
      signal: ac.signal,
      credentials: 'omit',
      headers: { 'Content-Type': 'application/json' },
      // 后端从 ClientIP 读 IP，不需要 body；保留空对象方便后续扩展
      body: '{}',
    })
    if (!resp.ok) {
      if (import.meta.env.DEV) {
        console.info('[edr] backend non-200', {
          status: resp.status,
          elapsedMs: Math.round(performance.now() - startedAt),
        })
      }
      return 'fail'
    }
    const data = (await resp.json()) as EDRCheckResponse
    const result: EdrResult = data.pass ? 'pass' : 'fail'
    if (import.meta.env.DEV) {
      console.info('[edr]', result, {
        url,
        reason: data.reason,
        elapsedMs: Math.round(performance.now() - startedAt),
      })
    }
    return result
  } catch (err) {
    if (import.meta.env.DEV) {
      console.info('[edr] fail (network)', {
        url,
        elapsedMs: Math.round(performance.now() - startedAt),
        err: err instanceof Error ? err.name : String(err),
      })
    }
    return 'fail'
  } finally {
    clearTimeout(timer)
  }
}
