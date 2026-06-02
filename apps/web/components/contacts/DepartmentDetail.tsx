"use client";

import { useEffect, useState } from "react";
import type { DepartmentDetail as Detail, EntityRef } from "@/lib/types";

// 部门详情卡片：路径面包屑(可点) + 数量指标 + 子部门 + 直属成员。
// 充分发挥图特性：recursive_member_count 走 PARENT_OF* + MEMBER_OF 一次聚合。
export function DepartmentDetail({
  id,
  onSelectEntity,
}: {
  id: string;
  onSelectEntity: (e: EntityRef) => void;
}) {
  const [data, setData] = useState<Detail | null | "loading" | "error">("loading");

  useEffect(() => {
    setData("loading");
    fetch(`/api/proxy/departments/${encodeURIComponent(id)}`)
      .then(async (r) => {
        if (r.status === 404) return null;
        if (!r.ok) throw new Error(String(r.status));
        return (await r.json()) as Detail;
      })
      .then((d) => setData(d))
      .catch(() => setData("error"));
  }, [id]);

  if (data === "loading") return <div className="text-sm text-zinc-400">加载部门...</div>;
  if (data === "error") return <div className="text-sm text-red-500">加载失败</div>;
  if (data === null) return <div className="text-sm text-zinc-400">部门不存在</div>;

  return (
    <div className="space-y-6">
      {/* 头部 */}
      <div>
        <div className="flex items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-gradient-to-br from-emerald-400 to-emerald-600 text-base font-semibold text-white">
            {data.name?.slice(0, 1) || "?"}
          </div>
          <div>
            <div className="text-base font-semibold">{data.name}</div>
            <div className="font-mono text-xs text-zinc-500">{data.department_id}</div>
          </div>
        </div>

        {/* 路径面包屑(可点击祖先) */}
        {data.path && data.path.length > 1 && (
          <div className="mt-3 flex flex-wrap items-center gap-1 text-xs text-zinc-500">
            {data.path.slice(0, -1).map((p) => (
              <span key={p.department_id} className="flex items-center">
                <button
                  type="button"
                  onClick={() => onSelectEntity({ type: "department", id: p.department_id })}
                  className="hover:text-blue-600 hover:underline dark:hover:text-blue-400"
                >
                  {p.name}
                </button>
                <span className="mx-1 text-zinc-300">/</span>
              </span>
            ))}
            <span className="font-medium text-zinc-700 dark:text-zinc-300">
              {data.path[data.path.length - 1].name}
            </span>
          </div>
        )}
      </div>

      {/* 数量指标 */}
      <div className="grid grid-cols-2 gap-3">
        <Metric label="直属成员" value={data.member_count} />
        <Metric label="含子部门总人数" value={data.recursive_member_count} hint="图聚合" />
      </div>

      {/* 子部门(可点跳转部门详情) */}
      <Section title={`子部门 (${data.children?.length ?? 0})`}>
        {data.children?.length ? (
          <div className="grid grid-cols-2 gap-2">
            {data.children.map((c) => (
              <button
                key={c.department_id}
                type="button"
                onClick={() => onSelectEntity({ type: "department", id: c.department_id })}
                className="rounded border border-zinc-200 px-3 py-2 text-left hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800"
              >
                <div className="text-sm font-medium">{c.name}</div>
                <div className="text-xs text-zinc-500">{c.member_count} 人</div>
              </button>
            ))}
          </div>
        ) : (
          <div className="text-xs text-zinc-400">无子部门</div>
        )}
      </Section>

      {/* 直属成员(可点跳转用户详情) */}
      <Section title={`直属成员 (${data.direct_members?.length ?? 0})`}>
        {data.direct_members?.length ? (
          <div className="grid grid-cols-2 gap-2">
            {data.direct_members.map((u) => (
              <button
                key={u.id}
                type="button"
                onClick={() => onSelectEntity({ type: "user", id: u.feishu_id })}
                className="rounded border border-zinc-200 px-3 py-2 text-left hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800"
              >
                <div className="text-sm font-medium">{u.name}</div>
                <div className="truncate text-xs text-zinc-500">{u.email || "—"}</div>
              </button>
            ))}
          </div>
        ) : (
          <div className="text-xs text-zinc-400">无直属成员</div>
        )}
      </Section>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500">{title}</div>
      {children}
    </div>
  );
}

function Metric({ label, value, hint }: { label: string; value: number; hint?: string }) {
  return (
    <div className="rounded-lg border border-zinc-200 px-3 py-2 dark:border-zinc-700">
      <div className="text-xs text-zinc-500">
        {label}
        {hint && <span className="ml-1 text-[10px] text-emerald-600">·{hint}</span>}
      </div>
      <div className="text-xl font-semibold">{value}</div>
    </div>
  );
}
