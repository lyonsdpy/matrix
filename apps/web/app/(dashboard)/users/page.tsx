import Link from "next/link";
import { serverFetch } from "@/lib/api";
import type { SyncedUserList } from "@/lib/types";
import { UserSearch } from "./user-search";

// 用户管理列表页：Server Component，按 URL searchParams 调 Go API 取数。
export default async function UsersPage({
  searchParams,
}: {
  searchParams: Promise<{ search?: string; cursor?: string }>;
}) {
  const { search = "", cursor = "" } = await searchParams;

  const qs = new URLSearchParams({ limit: "20" });
  if (search) qs.set("search", search);
  if (cursor) qs.set("cursor", cursor);

  const res = await serverFetch(`/api/v1/users?${qs.toString()}`);
  if (!res.ok) {
    return (
      <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
        加载用户失败（{res.status}）
      </div>
    );
  }
  const data: SyncedUserList = await res.json();

  // 下一页链接：携带当前搜索词 + 服务端返回的游标
  const nextQs = new URLSearchParams();
  if (search) nextQs.set("search", search);
  if (data.end_cursor) nextQs.set("cursor", data.end_cursor);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">用户管理</h1>
        <UserSearch />
      </div>

      <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <table className="w-full text-sm">
          <thead className="border-b border-zinc-200 bg-zinc-50 text-left text-xs uppercase tracking-wide text-zinc-500 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-400">
            <tr>
              <th className="px-4 py-3 font-medium">姓名</th>
              <th className="px-4 py-3 font-medium">open_id</th>
              <th className="px-4 py-3 font-medium">邮箱</th>
              <th className="px-4 py-3 font-medium">飞书 user_id</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800">
            {data.users.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-4 py-10 text-center text-zinc-400">
                  没有匹配的用户
                </td>
              </tr>
            ) : (
              data.users.map((u) => (
                <tr key={u.id} className="hover:bg-zinc-50 dark:hover:bg-zinc-800/50">
                  <td className="px-4 py-3 font-medium">{u.name}</td>
                  <td className="px-4 py-3 font-mono text-xs text-zinc-500">{u.feishu_id}</td>
                  <td className="px-4 py-3 text-zinc-600 dark:text-zinc-400">{u.email || "—"}</td>
                  <td className="px-4 py-3 font-mono text-xs text-zinc-500">{u.user_id || "—"}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <div className="flex justify-end">
        {data.has_next ? (
          <Link
            href={`/users?${nextQs.toString()}`}
            className="rounded-lg border border-zinc-300 px-4 py-2 text-sm font-medium transition-colors hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
          >
            下一页 →
          </Link>
        ) : (
          <span className="px-4 py-2 text-sm text-zinc-400">已到末页</span>
        )}
      </div>
    </div>
  );
}
