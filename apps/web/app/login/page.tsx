import { LoginTabs } from "./LoginTabs";

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

        <LoginTabs />
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
