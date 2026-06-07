import { useNavigate } from 'react-router-dom'

export function FailedPage() {
  const navigate = useNavigate()

  const handleRetry = () => {
    navigate({ pathname: '/check', search: window.location.search }, { replace: true })
  }

  const handleReload = () => {
    window.location.reload()
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-6">
      <div className="w-full max-w-md rounded-2xl bg-white p-8 shadow-sm ring-1 ring-slate-200">
        <div className="flex items-start gap-3">
          <span
            aria-hidden
            className="mt-0.5 inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-amber-100 text-amber-600"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="currentColor"
              className="h-5 w-5"
            >
              <path
                fillRule="evenodd"
                d="M12 2.25c.41 0 .79.22 1 .58l9.5 16.5a1.15 1.15 0 0 1-1 1.72H2.5a1.15 1.15 0 0 1-1-1.72L11 2.83c.21-.36.59-.58 1-.58Zm0 5.5a.94.94 0 0 0-.94 1.03l.38 4.97a.56.56 0 0 0 1.12 0l.38-4.97A.94.94 0 0 0 12 7.75Zm0 8.25a1 1 0 1 0 0 2 1 1 0 0 0 0-2Z"
                clipRule="evenodd"
              />
            </svg>
          </span>
          <div>
            <h1 className="text-lg font-semibold text-slate-900">终端安全检查未通过</h1>
            <p className="mt-3 text-sm leading-6 text-slate-600">
              未检测到企业安全客户端。请安装或启动：
            </p>
            <p className="mt-1 text-sm font-medium text-slate-900">亚信安全 OfficeScan Client</p>
            <p className="mt-3 text-sm leading-6 text-slate-600">
              完成后点击下方按钮重新检测。
            </p>
          </div>
        </div>

        <div className="mt-6 flex gap-3">
          <button
            type="button"
            onClick={handleRetry}
            className="flex-1 rounded-lg bg-slate-900 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800 focus:outline-none focus:ring-2 focus:ring-slate-900 focus:ring-offset-2"
          >
            重新检测
          </button>
          <button
            type="button"
            onClick={handleReload}
            className="flex-1 rounded-lg bg-white px-4 py-2.5 text-sm font-medium text-slate-700 ring-1 ring-inset ring-slate-300 transition hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-slate-400 focus:ring-offset-2"
          >
            刷新页面
          </button>
        </div>
      </div>
    </main>
  )
}
