"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

// 登录入口 Tab 切换：飞书扫码（默认） / 账号密码。
// 飞书扫码是日常通道；账号密码主要给 admin 在飞书故障时兜底使用。
export function LoginTabs() {
  const [tab, setTab] = useState<"lark" | "password">("lark");
  return (
    <div>
      <div className="mb-6 flex rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800">
        <TabButton active={tab === "lark"} onClick={() => setTab("lark")}>
          飞书扫码
        </TabButton>
        <TabButton active={tab === "password"} onClick={() => setTab("password")}>
          账号密码
        </TabButton>
      </div>
      {tab === "lark" ? <LarkLogin /> : <PasswordLogin />}
    </div>
  );
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={
        "flex-1 rounded-md px-3 py-1.5 text-sm font-medium transition-colors " +
        (active
          ? "bg-white text-zinc-900 shadow-sm dark:bg-zinc-700 dark:text-zinc-100"
          : "text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-200")
      }
    >
      {children}
    </button>
  );
}

function LarkLogin() {
  return (
    <a
      href="/api/auth/lark"
      className="flex w-full items-center justify-center gap-3 rounded-xl bg-blue-600 px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-blue-700"
    >
      <LarkIcon />
      飞书扫码登录
    </a>
  );
}

function PasswordLogin() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      const res = await fetch("/api/auth/password", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: username.trim(), password }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        setError(body?.error ?? "登录失败");
        return;
      }
      // 登录成功：cookie 已由后端写入，刷新到 dashboard
      // 用 router.replace + refresh 而非 location.href，让 Next 完成路由切换并刷新 server-side 数据
      router.replace("/contacts");
      router.refresh();
    } catch {
      setError("网络异常");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form className="space-y-3" onSubmit={submit}>
      <div>
        <label className="mb-1 block text-xs text-zinc-500">账号</label>
        <input
          type="text"
          autoComplete="username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="admin"
          className="w-full rounded-lg border border-zinc-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none dark:border-zinc-700 dark:bg-zinc-900"
        />
      </div>
      <div>
        <label className="mb-1 block text-xs text-zinc-500">密码</label>
        <input
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="w-full rounded-lg border border-zinc-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none dark:border-zinc-700 dark:bg-zinc-900"
        />
      </div>
      {error && (
        <div className="rounded-md bg-red-50 px-3 py-2 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-400">
          {error}
        </div>
      )}
      <button
        type="submit"
        disabled={submitting || !username || !password}
        className="w-full rounded-xl bg-blue-600 px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {submitting ? "登录中..." : "登录"}
      </button>
      <p className="text-center text-xs text-zinc-400">
        本地账号主要给管理员兜底使用，日常请用飞书扫码
      </p>
    </form>
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
