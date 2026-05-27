"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useState, useTransition } from "react";

// 搜索框：提交时把关键词写入 URL（?search=），由 Server Component 重新取数。
// 改变搜索会重置游标（不带 cursor），回到第一页。
export function UserSearch() {
  const router = useRouter();
  const params = useSearchParams();
  const [value, setValue] = useState(params.get("search") ?? "");
  const [pending, startTransition] = useTransition();

  function submit(e: React.FormEvent) {
    e.preventDefault();
    const q = new URLSearchParams();
    if (value.trim()) q.set("search", value.trim());
    startTransition(() => router.push(`/users?${q.toString()}`));
  }

  return (
    <form onSubmit={submit} className="flex gap-2">
      <input
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder="按姓名或邮箱搜索…"
        className="w-64 rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm outline-none focus:border-blue-500 dark:border-zinc-700 dark:bg-zinc-900"
      />
      <button
        type="submit"
        disabled={pending}
        className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
      >
        {pending ? "搜索中…" : "搜索"}
      </button>
    </form>
  );
}
