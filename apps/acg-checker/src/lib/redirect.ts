const FEISHU_AUTHORIZE_URL = 'https://accounts.feishu.cn/open-apis/authen/v1/authorize'

// D 自身的控制参数，不外泄给飞书
const D_CONTROL_KEYS = new Set(['force', 'redirect'])

// ACG 入口形如：/check?response_type=code&client_id=X&redirect_uri=<encoded(...&weburl=...&uplcyid=...)>
// 即 OAuth v2 标准透传，且 redirect_uri 内部已经由 ACG 嵌好 weburl/uplcyid。
// D 的职责仅是：把 client_id 改名为 app_id（飞书 v1 authorize 用 app_id），
// 把 redirect_uri 原样保留，连同其他 OAuth 参数一起转发给飞书 authorize 端点。
//
// 注意：必须原样保留 redirect_uri，否则飞书 token 阶段会校验失败（错误：
// "The provided redirect URI does not match the one used during authorization"）。
export function buildFeishuAuthorizeUrl(
  search: string = window.location.search,
): string | null {
  const trimmed = search.startsWith('?') ? search.slice(1) : search
  if (!trimmed) return null

  let params: URLSearchParams
  try {
    params = new URLSearchParams(trimmed)
  } catch {
    return null
  }

  // ACG 用 OAuth v2 标准命名 client_id；飞书 v1 authorize 端点用 app_id。
  // 兼容直接传 app_id 的情况（手工调用或老调用方）。
  const appId = params.get('client_id') ?? params.get('app_id')
  const redirectUri = params.get('redirect_uri')
  if (!appId || !redirectUri) return null

  const out = new URL(FEISHU_AUTHORIZE_URL)
  out.searchParams.set('app_id', appId)
  out.searchParams.set('redirect_uri', redirectUri)

  // 透传其余 OAuth 参数（state/scope/response_type 等），跳过 D 自身控制参数和已处理过的键
  for (const [k, v] of params.entries()) {
    if (D_CONTROL_KEYS.has(k)) continue
    if (k === 'client_id' || k === 'app_id' || k === 'redirect_uri') continue
    out.searchParams.set(k, v)
  }
  return out.toString()
}

// 兼容旧 MVP：?redirect=<encoded Feishu URL>。当前 ACG 集成不会走这条路径。
export function parseRedirectParam(
  search: string = window.location.search,
): string | null {
  try {
    const v = new URLSearchParams(search).get('redirect')
    return v || null
  } catch {
    return null
  }
}

export function goRedirect(url: string): void {
  window.location.href = url
}
