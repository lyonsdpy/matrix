import { describe, expect, it } from 'vitest'
import { parseRedirectParam } from './redirect'

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

  it('preserves additional query params inside the redirect value', () => {
    const target = 'https://x.com/cb?a=1&b=2'
    const search = `?redirect=${encodeURIComponent(target)}`
    expect(parseRedirectParam(search)).toBe(target)
  })
})

