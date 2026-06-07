import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { checkEdr } from './edr'

const API_BASE = 'http://api.example.com'
const ENDPOINT = `${API_BASE}/api/v1/acg/edr-check`

function jsonResponse(body: unknown, init: ResponseInit = {}): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
}

describe('checkEdr', () => {
  let fetchSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    vi.stubEnv('VITE_EDR_MOCK', '')
    vi.stubEnv('VITE_API_BASE_URL', API_BASE)
    fetchSpy = vi.spyOn(globalThis, 'fetch')
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllEnvs()
  })

  it("returns 'pass' when backend reports pass=true", async () => {
    fetchSpy.mockResolvedValue(jsonResponse({ pass: true }))
    const result = await checkEdr()
    expect(result).toBe('pass')
    expect(fetchSpy).toHaveBeenCalledOnce()
    const [url, init] = fetchSpy.mock.calls[0]!
    expect(url).toBe(ENDPOINT)
    expect(init?.method).toBe('POST')
  })

  it("returns 'fail' when backend reports pass=false", async () => {
    fetchSpy.mockResolvedValue(jsonResponse({ pass: false, reason: 'unexpected status 200' }))
    expect(await checkEdr()).toBe('fail')
  })

  it("returns 'fail' when backend returns non-2xx", async () => {
    fetchSpy.mockResolvedValue(new Response(null, { status: 500 }))
    expect(await checkEdr()).toBe('fail')
  })

  it("returns 'fail' when fetch rejects with TypeError (network error)", async () => {
    fetchSpy.mockRejectedValue(new TypeError('Failed to fetch'))
    expect(await checkEdr()).toBe('fail')
  })

  it("returns 'fail' when fetch aborts (timeout)", async () => {
    fetchSpy.mockImplementation(
      (_input: RequestInfo | URL, init?: RequestInit) =>
        new Promise<Response>((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () => {
            reject(new DOMException('aborted', 'AbortError'))
          })
        }),
    )
    const start = performance.now()
    const result = await checkEdr({ timeoutMs: 30 })
    const elapsed = performance.now() - start
    expect(result).toBe('fail')
    expect(elapsed).toBeGreaterThanOrEqual(25)
  })

  it("honors VITE_EDR_MOCK='pass' and skips fetch", async () => {
    vi.stubEnv('VITE_EDR_MOCK', 'pass')
    expect(await checkEdr()).toBe('pass')
    expect(fetchSpy).not.toHaveBeenCalled()
  })

  it("honors VITE_EDR_MOCK='fail' and skips fetch", async () => {
    vi.stubEnv('VITE_EDR_MOCK', 'fail')
    expect(await checkEdr()).toBe('fail')
    expect(fetchSpy).not.toHaveBeenCalled()
  })

  it('ignores unknown mock values and falls back to real fetch', async () => {
    vi.stubEnv('VITE_EDR_MOCK', 'maybe')
    fetchSpy.mockResolvedValue(jsonResponse({ pass: true }))
    expect(await checkEdr()).toBe('pass')
    expect(fetchSpy).toHaveBeenCalledOnce()
  })

  it('prefers opts.apiBaseUrl over env', async () => {
    fetchSpy.mockResolvedValue(jsonResponse({ pass: true }))
    await checkEdr({ apiBaseUrl: 'http://override.example.com/' })
    expect(fetchSpy.mock.calls[0]![0]).toBe('http://override.example.com/api/v1/acg/edr-check')
  })

  it('falls back to relative path when no apiBaseUrl configured', async () => {
    vi.stubEnv('VITE_API_BASE_URL', '')
    fetchSpy.mockResolvedValue(jsonResponse({ pass: true }))
    await checkEdr()
    expect(fetchSpy.mock.calls[0]![0]).toBe('/api/v1/acg/edr-check')
  })
})
