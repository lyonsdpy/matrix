import { describe, expect, it } from 'vitest'
import { buildFeishuAuthUrl } from './feishu'

describe('buildFeishuAuthUrl', () => {
  it('builds the canonical authorize URL with required params', () => {
    const url = buildFeishuAuthUrl({
      appId: 'cli_x',
      redirectUri: 'http://192.168.2.201:8000/oauth_auth_submit.php',
    })
    const parsed = new URL(url)
    expect(parsed.origin + parsed.pathname).toBe(
      'https://accounts.feishu.cn/open-apis/authen/v1/authorize',
    )
    expect(parsed.searchParams.get('app_id')).toBe('cli_x')
    expect(parsed.searchParams.get('redirect_uri')).toBe(
      'http://192.168.2.201:8000/oauth_auth_submit.php',
    )
    expect(parsed.searchParams.has('state')).toBe(false)
  })

  it('URL-encodes redirect_uri inside the query string', () => {
    const url = buildFeishuAuthUrl({
      appId: 'a',
      redirectUri: 'http://example.com:8000/path?x=1',
    })
    expect(url).toContain(
      'redirect_uri=http%3A%2F%2Fexample.com%3A8000%2Fpath%3Fx%3D1',
    )
  })

  it('includes state when provided', () => {
    const url = buildFeishuAuthUrl({
      appId: 'a',
      redirectUri: 'http://x',
      state: 'csrf-token-123',
    })
    expect(new URL(url).searchParams.get('state')).toBe('csrf-token-123')
  })

  it('omits state when undefined or empty string', () => {
    const a = buildFeishuAuthUrl({ appId: 'a', redirectUri: 'http://x', state: undefined })
    const b = buildFeishuAuthUrl({ appId: 'a', redirectUri: 'http://x', state: '' })
    expect(new URL(a).searchParams.has('state')).toBe(false)
    expect(new URL(b).searchParams.has('state')).toBe(false)
  })
})
