import { describe, expect, it } from 'vitest'
import { buildFeishuAuthorizeUrl, parseRedirectParam } from './redirect'

describe('parseRedirectParam', () => {
  it('returns the redirect value when present', () => {
    expect(parseRedirectParam('?redirect=https://x.com')).toBe('https://x.com')
  })

  it('returns null when redirect is missing', () => {
    expect(parseRedirectParam('?foo=bar')).toBeNull()
    expect(parseRedirectParam('')).toBeNull()
  })

  it('returns null for empty redirect value', () => {
    expect(parseRedirectParam('?redirect=')).toBeNull()
  })

  it('URL-decodes the redirect value automatically', () => {
    const encoded =
      'https%3A%2F%2Faccounts.feishu.cn%2Fopen-apis%2Fauthen%2Fv1%2Fauthorize%3Fapp_id%3Dx'
    expect(parseRedirectParam(`?redirect=${encoded}`)).toBe(
      'https://accounts.feishu.cn/open-apis/authen/v1/authorize?app_id=x',
    )
  })
})

describe('buildFeishuAuthorizeUrl', () => {
  const FEISHU = 'https://accounts.feishu.cn/open-apis/authen/v1/authorize'

  // ACG 跳到 D 的真实入口形态（含已嵌好业务参数的 redirect_uri）
  const acgEntrySearch = () => {
    const rd =
      'http://10.1.20.10:8000/oauth_auth_submit.php?action=oauth_login_code&weburl=http://success.secops.xlbsoft.com&uplcyid=1'
    return (
      `?response_type=code` +
      `&client_id=cli_a9b884b1e8b81bc8` +
      `&redirect_uri=${encodeURIComponent(rd)}`
    )
  }

  it('maps client_id to app_id and preserves redirect_uri verbatim', () => {
    const out = buildFeishuAuthorizeUrl(acgEntrySearch()) as string
    expect(out).toBeTruthy()
    const url = new URL(out)

    expect(url.origin + url.pathname).toBe(FEISHU)
    expect(url.searchParams.get('app_id')).toBe('cli_a9b884b1e8b81bc8')
    expect(url.searchParams.has('client_id')).toBe(false) // client_id 不应在飞书 URL 里出现

    // 关键：redirect_uri 必须原样保留，weburl/uplcyid/action 都还在
    const inner = new URL(url.searchParams.get('redirect_uri') as string)
    expect(inner.origin + inner.pathname).toBe('http://10.1.20.10:8000/oauth_auth_submit.php')
    expect(inner.searchParams.get('action')).toBe('oauth_login_code')
    expect(inner.searchParams.get('weburl')).toBe('http://success.secops.xlbsoft.com')
    expect(inner.searchParams.get('uplcyid')).toBe('1')
  })

  it('forwards response_type and other OAuth params through', () => {
    const url = new URL(buildFeishuAuthorizeUrl(acgEntrySearch()) as string)
    expect(url.searchParams.get('response_type')).toBe('code')
  })

  it('forwards state when present', () => {
    const rd = 'http://b/cb'
    const search = `?client_id=x&redirect_uri=${encodeURIComponent(rd)}&state=csrf-token-123`
    const url = new URL(buildFeishuAuthorizeUrl(search) as string)
    expect(url.searchParams.get('state')).toBe('csrf-token-123')
  })

  it('also accepts app_id directly (no client_id mapping needed)', () => {
    const rd = 'http://b/cb?a=1'
    const search = `?app_id=x&redirect_uri=${encodeURIComponent(rd)}`
    const url = new URL(buildFeishuAuthorizeUrl(search) as string)
    expect(url.searchParams.get('app_id')).toBe('x')
    expect(url.searchParams.get('redirect_uri')).toBe(rd)
  })

  it('strips D-only force / redirect params so they never leak to Feishu', () => {
    const rd = 'http://b/cb'
    const search = `?client_id=x&redirect_uri=${encodeURIComponent(rd)}&force=mac&redirect=ignored`
    const url = new URL(buildFeishuAuthorizeUrl(search) as string)
    expect(url.searchParams.has('force')).toBe(false)
    expect(url.searchParams.has('redirect')).toBe(false)
  })

  it('returns null when client_id/app_id is missing', () => {
    expect(buildFeishuAuthorizeUrl('?redirect_uri=http%3A%2F%2Fb%2Fcb')).toBeNull()
  })

  it('returns null when redirect_uri is missing', () => {
    expect(buildFeishuAuthorizeUrl('?client_id=x')).toBeNull()
    expect(buildFeishuAuthorizeUrl('?app_id=x')).toBeNull()
  })

  it('returns null on empty search', () => {
    expect(buildFeishuAuthorizeUrl('')).toBeNull()
  })

  it('tolerates a leading ? or its absence', () => {
    const rd = 'http://b/cb'
    const a = buildFeishuAuthorizeUrl(`?client_id=x&redirect_uri=${encodeURIComponent(rd)}`)
    const b = buildFeishuAuthorizeUrl(`client_id=x&redirect_uri=${encodeURIComponent(rd)}`)
    expect(a).toBe(b)
  })
})
