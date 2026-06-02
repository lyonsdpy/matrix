"use client";

import { useEffect, useState } from "react";
import type { ContactSearchResult, EntityRef } from "@/lib/types";

// 联合搜索：输入 debounce → /contacts/search → 下拉显示 用户 + 部门 两类结果。
// 点击任一结果都通过 onSelectEntity 触发对应详情组件(registry 模式)。
export function SearchBar({ onSelectEntity }: { onSelectEntity: (e: EntityRef) => void }) {
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);
  const [result, setResult] = useState<ContactSearchResult | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!q.trim()) {
      setResult(null);
      return;
    }
    const handle = setTimeout(async () => {
      setLoading(true);
      try {
        const data = await fetch(
          `/api/proxy/contacts/search?q=${encodeURIComponent(q.trim())}&limit=8`,
        ).then((r) => r.json());
        setResult(data);
      } catch {
        setResult({ users: [], departments: [] });
      } finally {
        setLoading(false);
      }
    }, 250);
    return () => clearTimeout(handle);
  }, [q]);

  const select = (e: EntityRef) => {
    onSelectEntity(e);
    setOpen(false);
  };

  const hasUsers = (result?.users?.length ?? 0) > 0;
  const hasDepts = (result?.departments?.length ?? 0) > 0;

  return (
    <div className="relative">
      <input
        type="search"
        value={q}
        onChange={(e) => {
          setQ(e.target.value);
          setOpen(true);
        }}
        onFocus={() => q && setOpen(true)}
        placeholder="搜索用户 / 部门..."
        className="w-full rounded-lg border border-zinc-300 bg-white px-3 py-1.5 text-sm placeholder:text-zinc-400 focus:border-blue-500 focus:outline-none dark:border-zinc-700 dark:bg-zinc-900"
      />
      {open && q && (
        <div className="absolute left-0 right-0 top-full z-40 mt-1 max-h-96 overflow-y-auto rounded-lg border border-zinc-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900">
          {loading && <div className="px-3 py-3 text-xs text-zinc-400">搜索中...</div>}
          {!loading && result && !hasUsers && !hasDepts && (
            <div className="px-3 py-3 text-xs text-zinc-400">无匹配结果</div>
          )}
          {hasDepts && (
            <div>
              <div className="px-3 pt-2 text-[10px] uppercase tracking-wider text-zinc-400">部门</div>
              {result!.departments.map((d) => (
                <button
                  key={d.department_id}
                  type="button"
                  onClick={() => select({ type: "department", id: d.department_id })}
                  className="block w-full px-3 py-2 text-left text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800"
                >
                  <span className="font-medium">{d.name}</span>
                  <span className="ml-2 text-xs text-zinc-400">{d.member_count} 人</span>
                </button>
              ))}
            </div>
          )}
          {hasUsers && (
            <div>
              <div className="px-3 pt-2 text-[10px] uppercase tracking-wider text-zinc-400">用户</div>
              {result!.users.map((u) => (
                <button
                  key={u.id}
                  type="button"
                  onClick={() => select({ type: "user", id: u.feishu_id })}
                  className="block w-full px-3 py-2 text-left text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800"
                >
                  <span className="font-medium">{u.name}</span>
                  <span className="ml-2 truncate text-xs text-zinc-400">
                    {(u.department_names ?? []).join("、") || u.email}
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
