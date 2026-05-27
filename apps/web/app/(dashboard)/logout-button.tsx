"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

// 登出：调 /api/auth/logout 清除 session cookie，然后跳登录页。
export function LogoutButton() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);

  async function handleLogout() {
    setLoading(true);
    await fetch("/api/auth/logout", { method: "POST" });
    router.replace("/login");
    router.refresh();
  }

  return (
    <button
      onClick={handleLogout}
      disabled={loading}
      className="rounded-lg px-3 py-1.5 text-sm font-medium text-zinc-600 transition-colors hover:bg-zinc-100 disabled:opacity-50 dark:text-zinc-400 dark:hover:bg-zinc-800"
    >
      {loading ? "登出中…" : "登出"}
    </button>
  );
}
