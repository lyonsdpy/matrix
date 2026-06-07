/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_EDR_MOCK?: 'pass' | 'fail'
  readonly VITE_FEISHU_APP_ID?: string
  readonly VITE_FEISHU_REDIRECT_URI?: string
  readonly VITE_FEISHU_STATE?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
