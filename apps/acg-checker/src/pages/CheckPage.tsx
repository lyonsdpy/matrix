import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Spinner } from '../components/Spinner'
import { checkEdr } from '../lib/edr'
import { buildDevFeishuAuthUrl } from '../lib/feishu'
import { goRedirect, parseRedirectParam } from '../lib/redirect'
import { detectRuntime } from '../lib/runtime'

type CheckState =
  | { kind: 'missing-redirect' }
  | { kind: 'detecting' }
  | { kind: 'redirecting' }

function resolveRedirect(search: string): string | null {
  const fromParam = parseRedirectParam(search)
  if (fromParam) return fromParam
  const devFallback = buildDevFeishuAuthUrl()
  if (devFallback && import.meta.env.DEV) {
    console.info('[redirect] using dev fallback Feishu auth URL', devFallback)
  }
  return devFallback
}

export function CheckPage() {
  const navigate = useNavigate()
  const [state, setState] = useState<CheckState>(() => {
    const redirect = resolveRedirect(window.location.search)
    return redirect ? { kind: 'detecting' } : { kind: 'missing-redirect' }
  })

  useEffect(() => {
    if (state.kind !== 'detecting') return

    const search = window.location.search
    const redirect = resolveRedirect(search)
    if (!redirect) {
      setState({ kind: 'missing-redirect' })
      return
    }

    let cancelled = false

    void (async () => {
      const runtime = detectRuntime()

      // 仅移动端免检直接放行；Mac 与 Windows 均需走 EDR 检测（后端探测
      // 客户端 8445 端口的安全客户端 both_way/communication 接口）。
      if (runtime.isMobile) {
        if (cancelled) return
        setState({ kind: 'redirecting' })
        goRedirect(redirect)
        return
      }

      const result = await checkEdr()
      if (cancelled) return

      if (result === 'pass') {
        setState({ kind: 'redirecting' })
        goRedirect(redirect)
      } else {
        navigate({ pathname: '/failed', search }, { replace: true })
      }
    })()

    return () => {
      cancelled = true
    }
  }, [navigate, state.kind])

  if (state.kind === 'missing-redirect') {
    return (
      <main className="flex min-h-screen items-center justify-center px-6">
        <div className="w-full max-w-md rounded-2xl bg-white p-8 text-center shadow-sm ring-1 ring-slate-200">
          <h1 className="text-lg font-semibold text-slate-900">无法开始检查</h1>
          <p className="mt-3 text-sm leading-6 text-slate-600">
            参数 <code className="rounded bg-slate-100 px-1.5 py-0.5 text-slate-800">redirect</code>{' '}
            缺失，请联系管理员。
          </p>
        </div>
      </main>
    )
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-6">
      <div className="flex flex-col items-center text-center">
        <Spinner />
        <p className="mt-6 text-base text-slate-700">正在检查终端安全状态...</p>
      </div>
    </main>
  )
}
