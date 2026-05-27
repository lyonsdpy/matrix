export default function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50 dark:bg-black">
      <div className="w-full max-w-sm rounded-2xl bg-white p-8 shadow-sm dark:bg-zinc-900">
        <h1 className="mb-8 text-center text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
          Matrix
        </h1>

        <ErrorBanner searchParams={searchParams} />

        {/* 飞书扫码登录：跳转 /api/auth/lark，由服务端生成授权 URL */}
        <a
          href="/api/auth/lark"
          className="flex w-full items-center justify-center gap-3 rounded-xl bg-blue-600 px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-blue-700"
        >
          <LarkIcon />
          飞书扫码登录
        </a>
      </div>
    </div>
  );
}

async function ErrorBanner({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const { error } = await searchParams;
  if (!error) return null;

  const messages: Record<string, string> = {
    invalid_state: "登录已过期或请求无效，请重试",
    auth_failed: "飞书认证失败，请重试",
  };

  return (
    <div className="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
      {messages[error] ?? "登录失败，请重试"}
    </div>
  );
}

function LarkIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 2C6.477 2 2 6.477 2 12s4.477 10 10 10 10-4.477 10-10S17.523 2 12 2zm0 18c-4.411 0-8-3.589-8-8s3.589-8 8-8 8 3.589 8 8-3.589 8-8 8z" />
      <path d="M12 6a1 1 0 0 0-1 1v5H7a1 1 0 0 0 0 2h5a1 1 0 0 0 1-1V7a1 1 0 0 0-1-1z" />
    </svg>
  );
}
