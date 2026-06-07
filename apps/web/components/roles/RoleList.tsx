"use client";

import { useState } from "react";
import type { Role } from "@/lib/types";

interface Props {
  roles: Role[];
  selectedId: string;
  onSelect: (id: string) => void;
  onChanged: () => void;
}

// 左栏：角色列表 + 新建/编辑/删除。
// is_system=true 的角色（admin/viewer）：底色不同，不可删；
// 改名/描述仍允许（便于本地化显示名）
export function RoleList({ roles, selectedId, onSelect, onChanged }: Props) {
  const [editingMode, setEditingMode] = useState<"create" | { id: string } | null>(null);

  return (
    <>
      <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3 dark:border-zinc-800">
        <span className="text-xs font-semibold uppercase tracking-wide text-zinc-400">
          角色
        </span>
        <button
          type="button"
          onClick={() => setEditingMode("create")}
          className="rounded-md bg-blue-600 px-2 py-1 text-xs font-medium text-white hover:bg-blue-700"
        >
          + 新建
        </button>
      </div>

      <div className="flex-1 space-y-1 overflow-y-auto p-2">
        {roles.length === 0 ? (
          <div className="px-3 py-6 text-center text-xs text-zinc-400">暂无角色</div>
        ) : (
          roles.map((r) => {
            const active = r.id === selectedId;
            return (
              <button
                key={r.id}
                type="button"
                onClick={() => onSelect(r.id)}
                className={
                  "flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm transition-colors " +
                  (active
                    ? "bg-blue-50 text-blue-900 dark:bg-blue-950 dark:text-blue-200"
                    : "text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-800")
                }
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="truncate font-medium">{r.name}</span>
                    {r.is_system && (
                      <span className="rounded bg-zinc-200 px-1.5 py-0.5 text-[10px] text-zinc-600 dark:bg-zinc-700 dark:text-zinc-300">
                        系统
                      </span>
                    )}
                  </div>
                  <div className="truncate text-xs text-zinc-400">
                    {r.code} · {r.user_count} 人
                  </div>
                </div>
              </button>
            );
          })
        )}
      </div>

      {/* 选中角色的操作按钮（编辑/删除）放在底部，避免与上方"新建"按钮风格冲突 */}
      {selectedId && (
        <RoleActions
          role={roles.find((r) => r.id === selectedId)}
          onEdit={() => setEditingMode({ id: selectedId })}
          onChanged={onChanged}
        />
      )}

      {/* 弹窗：新建 / 编辑 */}
      {editingMode === "create" && (
        <RoleEditor
          mode="create"
          onClose={() => setEditingMode(null)}
          onSaved={() => {
            setEditingMode(null);
            onChanged();
          }}
        />
      )}
      {editingMode && editingMode !== "create" && (
        <RoleEditor
          mode="edit"
          role={roles.find((r) => r.id === editingMode.id)}
          onClose={() => setEditingMode(null)}
          onSaved={() => {
            setEditingMode(null);
            onChanged();
          }}
        />
      )}
    </>
  );
}

function RoleActions({
  role,
  onEdit,
  onChanged,
}: {
  role?: Role;
  onEdit: () => void;
  onChanged: () => void;
}) {
  if (!role) return null;
  const doDelete = async () => {
    if (!confirm(`确认删除角色「${role.name}」？该角色下的用户绑定将一并解除。`)) return;
    const res = await fetch(`/api/proxy/roles/${role.id}`, { method: "DELETE" });
    if (res.ok) {
      onChanged();
    } else {
      const body = await res.json().catch(() => ({}));
      alert(body?.msg ?? "删除失败");
    }
  };
  return (
    <div className="flex items-center gap-2 border-t border-zinc-200 px-3 py-2 dark:border-zinc-800">
      <button
        type="button"
        onClick={onEdit}
        className="flex-1 rounded-md border border-zinc-300 px-2 py-1 text-xs text-zinc-700 hover:bg-zinc-50 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
      >
        编辑
      </button>
      <button
        type="button"
        onClick={doDelete}
        disabled={role.is_system}
        className="flex-1 rounded-md border border-zinc-300 px-2 py-1 text-xs text-zinc-700 hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-40 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
        title={role.is_system ? "系统内置角色不可删除" : "删除角色"}
      >
        删除
      </button>
    </div>
  );
}

function RoleEditor({
  mode,
  role,
  onClose,
  onSaved,
}: {
  mode: "create" | "edit";
  role?: Role;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [code, setCode] = useState(role?.code ?? "");
  const [name, setName] = useState(role?.name ?? "");
  const [desc, setDesc] = useState(role?.description ?? "");
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState("");

  const submit = async () => {
    setSaving(true);
    setErr("");
    try {
      const url = mode === "create" ? "/api/proxy/roles" : `/api/proxy/roles/${role!.id}`;
      const method = mode === "create" ? "POST" : "PUT";
      const body =
        mode === "create"
          ? { code: code.trim(), name: name.trim(), description: desc.trim() }
          : { name: name.trim(), description: desc.trim() };
      const res = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setErr(data?.msg ?? "保存失败");
        return;
      }
      onSaved();
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-md rounded-xl bg-white p-5 shadow-xl dark:bg-zinc-900">
        <h3 className="mb-4 text-base font-semibold">
          {mode === "create" ? "新建角色" : "编辑角色"}
        </h3>
        <div className="space-y-3 text-sm">
          <Field label="Code（机器名）">
            <input
              type="text"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              disabled={mode === "edit"}
              placeholder="小写字母/数字/下划线，2-64 字符；创建后不可改"
              className="w-full rounded-md border border-zinc-300 px-2 py-1.5 text-sm disabled:bg-zinc-100 disabled:text-zinc-500 dark:border-zinc-700 dark:bg-zinc-900 dark:disabled:bg-zinc-800"
            />
          </Field>
          <Field label="名称">
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full rounded-md border border-zinc-300 px-2 py-1.5 text-sm dark:border-zinc-700 dark:bg-zinc-900"
            />
          </Field>
          <Field label="描述">
            <textarea
              rows={3}
              value={desc}
              onChange={(e) => setDesc(e.target.value)}
              className="w-full rounded-md border border-zinc-300 px-2 py-1.5 text-sm dark:border-zinc-700 dark:bg-zinc-900"
            />
          </Field>
          {err && <div className="text-xs text-red-600">{err}</div>}
        </div>
        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-zinc-300 px-3 py-1.5 text-sm dark:border-zinc-700"
          >
            取消
          </button>
          <button
            type="button"
            disabled={saving || !name.trim() || (mode === "create" && !code.trim())}
            onClick={submit}
            className="rounded-md bg-blue-600 px-3 py-1.5 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {saving ? "保存中..." : "保存"}
          </button>
        </div>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium text-zinc-600 dark:text-zinc-400">
        {label}
      </span>
      {children}
    </label>
  );
}
