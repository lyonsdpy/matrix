"use client";

import { useEffect, useMemo, useState } from "react";
import type { Permission, RoleDetail } from "@/lib/types";

interface Props {
  role: RoleDetail;
  permissions: Permission[];
  onSaved: () => void;
}

// 中栏：权限勾选树。
// - admin 角色：渲染占位"系统管理员，拥有所有权限"，不显示勾选树（中间件特判，无法配置）
// - 其他角色：按 module 分组，第一期只渲染 kind=page；action 节点已落库预留，未来需要时打开
//   下方"显示动作权限"开关可临时切到 page+action 视图（不强保存）
export function PermissionTree({ role, permissions, onSaved }: Props) {
  const isAdmin = role.code === "admin";
  const [showActions, setShowActions] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(() => new Set(role.permission_codes));
  const [saving, setSaving] = useState(false);

  // 选中角色变化时同步重置勾选状态
  useEffect(() => {
    setSelected(new Set(role.permission_codes));
  }, [role.id, role.permission_codes]);

  // 按 module 分组：[{module, pages: [{page, actions: [...]}]}]
  const grouped = useMemo(() => groupByModule(permissions), [permissions]);

  if (isAdmin) {
    return (
      <div className="flex flex-1 flex-col">
        <Header role={role} />
        <div className="flex flex-1 items-center justify-center px-6 text-center text-sm text-zinc-500">
          <div>
            <div className="mb-2 text-base font-medium text-zinc-700 dark:text-zinc-300">
              系统管理员
            </div>
            <div className="text-xs text-zinc-400">
              拥有所有权限（受中间件特判保护，无需也无法逐项配置）
            </div>
          </div>
        </div>
      </div>
    );
  }

  const toggle = (code: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(code)) next.delete(code);
      else next.add(code);
      return next;
    });
  };

  // 勾选 page 时自动勾选其全部 action 子项；取消 page 时连带取消子项
  const togglePage = (page: Permission, actions: Permission[]) => {
    setSelected((prev) => {
      const next = new Set(prev);
      const checked = next.has(page.code);
      if (checked) {
        next.delete(page.code);
        actions.forEach((a) => next.delete(a.code));
      } else {
        next.add(page.code);
        if (showActions) actions.forEach((a) => next.add(a.code));
      }
      return next;
    });
  };

  const save = async () => {
    setSaving(true);
    try {
      const res = await fetch(`/api/proxy/roles/${role.id}/permissions`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ codes: Array.from(selected) }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        alert(body?.msg ?? "保存失败");
        return;
      }
      onSaved();
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <Header role={role}>
        <label className="flex items-center gap-2 text-xs text-zinc-500">
          <input
            type="checkbox"
            checked={showActions}
            onChange={(e) => setShowActions(e.target.checked)}
          />
          显示动作权限（第二期）
        </label>
      </Header>

      <div className="flex-1 space-y-4 overflow-y-auto p-4">
        {grouped.map((g) => (
          <div key={g.module} className="rounded-lg border border-zinc-200 dark:border-zinc-800">
            <div className="border-b border-zinc-100 bg-zinc-50 px-3 py-2 text-xs font-semibold uppercase tracking-wide text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900">
              {g.module}
            </div>
            <div className="space-y-1 p-2">
              {g.pages.map((p) => {
                const actions = showActions ? p.actions : [];
                return (
                  <div key={p.page.code}>
                    <label className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-zinc-50 dark:hover:bg-zinc-800">
                      <input
                        type="checkbox"
                        checked={selected.has(p.page.code)}
                        onChange={() => togglePage(p.page, p.actions)}
                      />
                      <span className="text-sm font-medium">{p.page.name}</span>
                      <span className="text-xs text-zinc-400">{p.page.code}</span>
                    </label>
                    {actions.map((a) => (
                      <label
                        key={a.code}
                        className="ml-6 flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-zinc-50 dark:hover:bg-zinc-800"
                      >
                        <input
                          type="checkbox"
                          checked={selected.has(a.code)}
                          onChange={() => toggle(a.code)}
                        />
                        <span className="text-sm text-zinc-600 dark:text-zinc-400">{a.name}</span>
                        <span className="text-xs text-zinc-400">{a.code}</span>
                      </label>
                    ))}
                  </div>
                );
              })}
            </div>
          </div>
        ))}
      </div>

      <div className="flex justify-end gap-2 border-t border-zinc-200 px-4 py-3 dark:border-zinc-800">
        <button
          type="button"
          onClick={() => setSelected(new Set(role.permission_codes))}
          className="rounded-md border border-zinc-300 px-3 py-1.5 text-sm dark:border-zinc-700"
        >
          重置
        </button>
        <button
          type="button"
          disabled={saving}
          onClick={save}
          className="rounded-md bg-blue-600 px-3 py-1.5 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {saving ? "保存中..." : "保存权限"}
        </button>
      </div>
    </div>
  );
}

function Header({ role, children }: { role: RoleDetail; children?: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3 dark:border-zinc-800">
      <div className="flex items-center gap-3">
        <div className="text-sm font-semibold">{role.name}</div>
        <div className="text-xs text-zinc-400">{role.code}</div>
      </div>
      {children}
    </div>
  );
}

// 按 module 分组，再把 action 挂到所属 page 下
function groupByModule(perms: Permission[]) {
  const byModule = new Map<string, { page: Permission; actions: Permission[] }[]>();
  const pageMap = new Map<string, { page: Permission; actions: Permission[] }>();

  // 先放 page
  for (const p of perms) {
    if (p.kind === "page") {
      const entry = { page: p, actions: [] as Permission[] };
      pageMap.set(p.code, entry);
      const arr = byModule.get(p.module) ?? [];
      arr.push(entry);
      byModule.set(p.module, arr);
    }
  }
  // 再挂 action
  for (const p of perms) {
    if (p.kind === "action" && p.parent_code) {
      const entry = pageMap.get(p.parent_code);
      if (entry) entry.actions.push(p);
    }
  }
  return Array.from(byModule.entries()).map(([module, pages]) => ({ module, pages }));
}
