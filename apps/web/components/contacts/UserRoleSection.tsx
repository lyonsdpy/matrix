"use client";

import { useCallback, useEffect, useState } from "react";
import type { Role } from "@/lib/types";

// 通讯录用户详情中的"系统角色"区块。
// 用 lark open_id 调 /auth-users/by-lark/:openId 反查本地账号：
//   - 404 → 用户不在 PG users 白名单，无法登录系统，渲染"未授权"占位
//   - 200 → 拿到本地 user_id，进一步取其角色 chips；带"编辑授权"按钮
interface AuthUserInfo {
  id: string;
  username: string;
  lark_open_id: string;
}

export function UserRoleSection({ openID }: { openID: string }) {
  const [state, setState] = useState<
    | { kind: "loading" }
    | { kind: "not-authorized" }
    | { kind: "ok"; user: AuthUserInfo; roles: Role[] }
    | { kind: "error" }
  >({ kind: "loading" });
  const [editing, setEditing] = useState(false);

  const reload = useCallback(async () => {
    setState({ kind: "loading" });
    try {
      const r = await fetch(`/api/proxy/auth-users/by-lark/${encodeURIComponent(openID)}`);
      if (r.status === 404) {
        setState({ kind: "not-authorized" });
        return;
      }
      if (!r.ok) {
        setState({ kind: "error" });
        return;
      }
      const user: AuthUserInfo = await r.json();
      const rolesResp = await fetch(`/api/proxy/auth-users/${user.id}/roles`).then((x) => x.json());
      setState({ kind: "ok", user, roles: rolesResp.items ?? [] });
    } catch {
      setState({ kind: "error" });
    }
  }, [openID]);

  useEffect(() => {
    reload();
  }, [reload]);

  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <div className="text-xs font-semibold uppercase tracking-wider text-zinc-500">
          系统角色
        </div>
        {state.kind === "ok" && (
          <button
            type="button"
            onClick={() => setEditing(true)}
            className="text-xs text-blue-600 hover:underline"
          >
            编辑授权
          </button>
        )}
      </div>

      {state.kind === "loading" && (
        <div className="text-xs text-zinc-400">加载中...</div>
      )}
      {state.kind === "error" && (
        <div className="text-xs text-red-500">加载失败</div>
      )}
      {state.kind === "not-authorized" && (
        <div className="rounded border border-dashed border-zinc-300 px-3 py-2 text-xs text-zinc-500 dark:border-zinc-700">
          该用户尚未在系统授权白名单中（无法登录本系统）
        </div>
      )}
      {state.kind === "ok" && (
        <>
          <div className="mb-1 text-xs text-zinc-500">
            本地账号：<span className="font-mono">{state.user.username}</span>
          </div>
          {state.roles.length === 0 ? (
            <div className="text-xs text-zinc-400">未绑定任何角色（登录后无权限）</div>
          ) : (
            <div className="flex flex-wrap gap-1">
              {state.roles.map((r) => (
                <span
                  key={r.id}
                  className={
                    "rounded-full px-2 py-0.5 text-xs " +
                    (r.is_system
                      ? "bg-zinc-200 text-zinc-700 dark:bg-zinc-700 dark:text-zinc-200"
                      : "bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-200")
                  }
                  title={r.code}
                >
                  {r.name}
                </span>
              ))}
            </div>
          )}
        </>
      )}

      {editing && state.kind === "ok" && (
        <EditAuthDialog
          user={state.user}
          currentRoles={state.roles}
          onClose={() => setEditing(false)}
          onSaved={() => {
            setEditing(false);
            reload();
          }}
        />
      )}
    </div>
  );
}

function EditAuthDialog({
  user,
  currentRoles,
  onClose,
  onSaved,
}: {
  user: AuthUserInfo;
  currentRoles: Role[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const [allRoles, setAllRoles] = useState<Role[]>([]);
  const [selected, setSelected] = useState<Set<string>>(
    () => new Set(currentRoles.map((r) => r.id)),
  );
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    fetch("/api/proxy/roles")
      .then((r) => r.json())
      .then((d) => setAllRoles(d.items ?? []));
  }, []);

  const toggle = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const submit = async () => {
    setSaving(true);
    try {
      const res = await fetch(`/api/proxy/auth-users/${user.id}/roles`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ role_ids: Array.from(selected) }),
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-md rounded-xl bg-white p-5 shadow-xl dark:bg-zinc-900">
        <h3 className="mb-1 text-base font-semibold">编辑授权</h3>
        <div className="mb-4 text-xs text-zinc-500">
          为 <span className="font-mono">{user.username}</span> 分配系统角色
        </div>
        <div className="max-h-72 space-y-1 overflow-y-auto rounded-md border border-zinc-200 p-2 dark:border-zinc-800">
          {allRoles.length === 0 ? (
            <div className="px-2 py-4 text-center text-xs text-zinc-400">无可选角色</div>
          ) : (
            allRoles.map((r) => (
              <label
                key={r.id}
                className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-zinc-50 dark:hover:bg-zinc-800"
              >
                <input
                  type="checkbox"
                  checked={selected.has(r.id)}
                  onChange={() => toggle(r.id)}
                />
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-medium">
                    {r.name}
                    {r.is_system && (
                      <span className="ml-1 rounded bg-zinc-200 px-1 text-[10px] text-zinc-600 dark:bg-zinc-700 dark:text-zinc-300">
                        系统
                      </span>
                    )}
                  </div>
                  <div className="truncate text-xs text-zinc-400">{r.code}</div>
                </div>
              </label>
            ))
          )}
        </div>
        <div className="mt-4 flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-zinc-300 px-3 py-1.5 text-sm dark:border-zinc-700"
          >
            取消
          </button>
          <button
            type="button"
            disabled={saving}
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
