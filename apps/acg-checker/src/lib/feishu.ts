const AUTHORIZE_URL = 'https://accounts.feishu.cn/open-apis/authen/v1/authorize'

export interface FeishuAuthParams {
  appId: string
  redirectUri: string
  state?: string
}

export function buildFeishuAuthUrl({ appId, redirectUri, state }: FeishuAuthParams): string {
  const params = new URLSearchParams({
    app_id: appId,
    redirect_uri: redirectUri,
  })
  if (state) params.set('state', state)
  return `${AUTHORIZE_URL}?${params.toString()}`
}

export function buildDevFeishuAuthUrl(): string | null {
  if (!import.meta.env.DEV) return null
  const appId = import.meta.env.VITE_FEISHU_APP_ID
  const redirectUri = import.meta.env.VITE_FEISHU_REDIRECT_URI
  if (!appId || !redirectUri) return null
  return buildFeishuAuthUrl({
    appId,
    redirectUri,
    state: import.meta.env.VITE_FEISHU_STATE,
  })
}
