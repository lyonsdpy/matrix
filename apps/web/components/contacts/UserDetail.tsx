"use client";

import { useEffect, useState } from "react";
import type { EntityRef, UserDetailData } from "@/lib/types";
import { UserRoleSection } from "./UserRoleSection";

// 用户详情卡片：基本信息 + 所属部门(带完整路径) + 同事(可点击切换抽屉)。
// onSelectEntity 让卡片内的部门/同事跳转到对应实体详情(EntityDrawer 内切换)。
export function UserDetail({
  id,
  onSelectEntity,
}: {
  id: string;
  onSelectEntity: (e: EntityRef) => void;
}) {
  const [data, setData] = useState<UserDetailData | null | "loading" | "error">("loading");

  useEffect(() => {
    setData("loading");
    fetch(`/api/proxy/users/${encodeURIComponent(id)}`)
      .then(async (r) => {
        if (r.status === 404) return null;
        if (!r.ok) throw new Error(String(r.status));
        return (await r.json()) as UserDetailData;
      })
      .then((d) => setData(d))
      .catch(() => setData("error"));
  }, [id]);

  if (data === "loading") return <div className="text-sm text-zinc-400">加载用户...</div>;
  if (data === "error") return <div className="text-sm text-red-500">加载失败</div>;
  if (data === null) return <div className="text-sm text-zinc-400">用户不存在</div>;

  return (
    <div className="space-y-6">
      {/* 基本信息 */}
      <div>
        <div className="flex items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-gradient-to-br from-blue-400 to-blue-600 text-base font-semibold text-white">
            {data.name?.slice(0, 1) || "?"}
          </div>
          <div>
            <div className="text-base font-semibold">{data.name}</div>
            <div className="text-xs text-zinc-500">
              {statusLabel(data.status)}
            </div>
          </div>
        </div>
        <div className="mt-4 grid grid-cols-[80px_1fr] gap-y-2 text-sm">
          <span className="text-zinc-500">邮箱</span>
          <span className="break-all">{data.email || "—"}</span>
          <span className="text-zinc-500">user_id</span>
          <span className="font-mono text-xs text-zinc-600 break-all">{data.user_id || "—"}</span>
          <span className="text-zinc-500">open_id</span>
          <span className="font-mono text-xs text-zinc-600 break-all">{data.feishu_id}</span>
        </div>
      </div>

      {/* 所属部门(每个带完整路径，可点击) */}
      <Section title={`所属部门 (${data.departments?.length ?? 0})`}>
        <div className="space-y-2">
          {(data.departments ?? []).map((d) => (
            <div
              key={d.department_id}
              className="rounded-lg border border-zinc-200 px-3 py-2 dark:border-zinc-700"
            >
              <button
                type="button"
                onClick={() => onSelectEntity({ type: "department", id: d.department_id })}
                className="text-sm font-medium text-blue-600 hover:underline dark:text-blue-400"
              >
                {d.name}
              </button>
              {d.path && d.path.length > 0 && (
                <div className="mt-1 flex flex-wrap items-center gap-1 text-xs text-zinc-500">
                  {d.path.map((p, idx) => (
                    <span key={p.department_id} className="flex items-center">
                      {idx > 0 && <span className="mx-1 text-zinc-300">/</span>}
                      <button
                        type="button"
                        onClick={() => onSelectEntity({ type: "department", id: p.department_id })}
                        className="hover:text-blue-600 hover:underline dark:hover:text-blue-400"
                      >
                        {p.name}
                      </button>
                    </span>
                  ))}
                </div>
              )}
            </div>
          ))}
          {(data.departments?.length ?? 0) === 0 && (
            <div className="text-xs text-zinc-400">无部门归属</div>
          )}
        </div>
      </Section>

      {/* 上级领导(图特性：沿 PARENT_OF 链向上找首个非自己的 leader) */}
      <Section title={`上级领导 (${data.leaders?.length ?? 0})`}>
        <div className="grid grid-cols-2 gap-2">
          {(data.leaders ?? []).map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => onSelectEntity({ type: "user", id: p.feishu_id })}
              className="rounded border border-zinc-200 px-3 py-2 text-left hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800"
            >
              <div className="text-sm font-medium">{p.name}</div>
              <div className="truncate text-xs text-zinc-500">
                {(p.department_names ?? []).join("、") || p.email || "—"}
              </div>
            </button>
          ))}
          {(data.leaders?.length ?? 0) === 0 && (
            <div className="col-span-2 text-xs text-zinc-400">暂无上级</div>
          )}
        </div>
      </Section>

      {/* 管理部门(本人作为 leader_user_id 的部门) */}
      <Section title={`管理部门 (${data.managed_departments?.length ?? 0})`}>
        <div className="flex flex-wrap gap-2">
          {(data.managed_departments ?? []).map((d) => (
            <button
              key={d.department_id}
              type="button"
              onClick={() => onSelectEntity({ type: "department", id: d.department_id })}
              className="rounded-full border border-zinc-300 px-3 py-1 text-xs hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800"
            >
              {d.name}
            </button>
          ))}
          {(data.managed_departments?.length ?? 0) === 0 && (
            <div className="text-xs text-zinc-400">未管理任何部门</div>
          )}
        </div>
      </Section>

      {/* 系统授权：白名单中的本地账号可编辑角色 */}
      <UserRoleSection openID={data.feishu_id} />
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500">
        {title}
      </div>
      {children}
    </div>
  );
}

function statusLabel(s: number): string {
  switch (s) {
    case 1:
      return "在职";
    case 2:
      return "已冻结";
    case 3:
      return "已离职";
    case 4:
      return "待入职";
    default:
      return "未知状态";
  }
}
