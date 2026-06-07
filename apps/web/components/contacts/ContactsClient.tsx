"use client";

import { useEffect, useState } from "react";
import type { DepartmentDetail, EntityRef } from "@/lib/types";
import { DepartmentTree } from "./DepartmentTree";
import { SearchBar } from "./SearchBar";
import { EntityDrawer } from "./EntityDrawer";

// 通讯录主交互：两栏布局 + 抽屉详情。
// 左：部门树(懒加载)+ 搜索框；中：当前选中部门的成员列表；右：抽屉(用户/部门切换)。
export function ContactsClient() {
  const [selectedDept, setSelectedDept] = useState<{ id: string; name: string } | null>(null);
  const [deptDetail, setDeptDetail] = useState<DepartmentDetail | null | "loading">(null);
  const [drawerEntity, setDrawerEntity] = useState<EntityRef | null>(null);

  // 选中部门变化 → 拉取直属成员预览(走和详情面板同一接口，已带 direct_members + recursive_count)
  useEffect(() => {
    if (!selectedDept) {
      setDeptDetail(null);
      return;
    }
    setDeptDetail("loading");
    fetch(`/api/proxy/departments/${encodeURIComponent(selectedDept.id)}`)
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => setDeptDetail(d))
      .catch(() => setDeptDetail(null));
  }, [selectedDept]);

  return (
    // 不再用 calc(100vh-7rem) 反推：父级 dashboard layout 已限定整页 h-screen，
    // 这里只需 flex-1 占满父容器剩余空间；min-h-0 让 flex 子项可以正确收缩，否则内部 overflow 会被忽略。
    <div className="flex min-h-0 flex-1 gap-4">
      {/* 左侧：搜索 + 部门树 */}
      <div className="flex w-80 shrink-0 flex-col gap-3 overflow-hidden rounded-xl border border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-900">
        <SearchBar onSelectEntity={setDrawerEntity} />
        <div className="-mx-3 flex-1 overflow-y-auto border-t border-zinc-100 pt-2 dark:border-zinc-800">
          <DepartmentTree
            onSelect={(id, name) => setSelectedDept({ id, name })}
            selectedId={selectedDept?.id ?? ""}
          />
        </div>
      </div>

      {/* 右侧：部门成员区 */}
      <div className="flex flex-1 flex-col overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        {!selectedDept ? (
          <EmptyState />
        ) : (
          <DeptMemberPanel
            selectedDept={selectedDept}
            detail={deptDetail}
            onSelectEntity={setDrawerEntity}
          />
        )}
      </div>

      {/* 抽屉：根据 entity.type 路由到 UserDetail / DepartmentDetail */}
      <EntityDrawer
        entity={drawerEntity}
        onClose={() => setDrawerEntity(null)}
        onSelectEntity={setDrawerEntity}
      />
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-2 text-sm text-zinc-400">
      <div className="text-4xl">📁</div>
      <div>从左侧选择部门查看成员</div>
      <div className="text-xs">或在搜索框输入用户/部门名直接定位</div>
    </div>
  );
}

function DeptMemberPanel({
  selectedDept,
  detail,
  onSelectEntity,
}: {
  selectedDept: { id: string; name: string };
  detail: DepartmentDetail | null | "loading";
  onSelectEntity: (e: EntityRef) => void;
}) {
  return (
    <>
      {/* 头部：面包屑 + 当前部门 + 查看详情 */}
      <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3 dark:border-zinc-800">
        <div className="min-w-0">
          {detail && detail !== "loading" && detail.path && detail.path.length > 1 && (
            <div className="mb-1 flex flex-wrap items-center gap-1 text-xs text-zinc-500">
              {detail.path.slice(0, -1).map((p) => (
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
            </div>
          )}
          <div className="flex items-center gap-2">
            <span className="truncate text-base font-semibold">{selectedDept.name}</span>
            {detail && detail !== "loading" && (
              <span className="text-xs text-zinc-500">
                · 直属 {detail.member_count} · 含子部门 {detail.recursive_member_count}
              </span>
            )}
          </div>
        </div>
        <button
          type="button"
          onClick={() => onSelectEntity({ type: "department", id: selectedDept.id })}
          className="shrink-0 rounded-lg border border-zinc-300 px-3 py-1.5 text-xs font-medium hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800"
        >
          查看部门详情
        </button>
      </div>

      {/* 内容区：子部门快捷入口 + 直属成员列表 */}
      <div className="flex-1 overflow-y-auto p-4">
        {detail === "loading" && <div className="text-sm text-zinc-400">加载中...</div>}
        {detail && detail !== "loading" && (
          <div className="space-y-6">
            {detail.children && detail.children.length > 0 && (
              <div>
                <div className="mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500">
                  子部门 ({detail.children.length})
                </div>
                <div className="flex flex-wrap gap-2">
                  {detail.children.map((c) => (
                    <button
                      key={c.department_id}
                      type="button"
                      onClick={() => onSelectEntity({ type: "department", id: c.department_id })}
                      className="rounded-full border border-zinc-300 px-3 py-1 text-xs hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800"
                    >
                      {c.name}
                      <span className="ml-1 text-zinc-400">{c.member_count}</span>
                    </button>
                  ))}
                </div>
              </div>
            )}

            <div>
              <div className="mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500">
                直属成员 ({detail.direct_members?.length ?? 0})
              </div>
              {detail.direct_members && detail.direct_members.length > 0 ? (
                <table className="w-full text-sm">
                  <thead className="border-b border-zinc-200 text-left text-xs uppercase tracking-wide text-zinc-500 dark:border-zinc-800">
                    <tr>
                      <th className="px-3 py-2 font-medium">姓名</th>
                      <th className="px-3 py-2 font-medium">邮箱</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800">
                    {detail.direct_members.map((u) => (
                      <tr
                        key={u.id}
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                        onClick={() => onSelectEntity({ type: "user", id: u.feishu_id })}
                      >
                        <td className="px-3 py-2 font-medium">{u.name}</td>
                        <td className="px-3 py-2 text-zinc-600 dark:text-zinc-400">{u.email || "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : (
                <div className="text-xs text-zinc-400">该部门无直属成员(成员可能都在子部门下)</div>
              )}
            </div>
          </div>
        )}
      </div>
    </>
  );
}
