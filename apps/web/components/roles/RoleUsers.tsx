"use client";

import { useCallback, useEffect, useState } from "react";
import type { Role, RoleUser, SyncedUser } from "@/lib/types";

interface Props {
  role: Role;
  onChanged: () => void;
}

// 右栏：该角色下的用户列表 + 添加/移除。
// 添加用户：弹出搜索框，从已同步的飞书用户中选；
// 添加后由后端把对应 PG 用户绑定到本角色（前提：该飞书用户已在 PG users 白名单内）
export function RoleUsers({ role, onChanged }: Props) {
  const [users, setUsers] = useState<RoleUser[]>([]);
  const [adding, setAdding] = useState(false);

  const reload = useCallback(async () => {
    const d = await fetch(`/api/proxy/roles/${role.id}/users`).then((r) => r.json());
    setUsers(d.items ?? []);
  }, [role.id]);

  useEffect(() => {
    reload();
  }, [reload]);

  const removeUser = async (userID: string, username: string) => {
    if (!confirm(`将「${username}」从角色「${role.name}」中移除？`)) return;
    const res = await fetch(`/api/proxy/roles/${role.id}/users/${userID}`, { method: "DELETE" });
    if (res.ok) {
      reload();
      onChanged();
    }
  };

  return (
    <>
      <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3 dark:border-zinc-800">
        <span className="text-xs font-semibold uppercase tracking-wide text-zinc-400">
          成员（{users.length}）
        </span>
        <button
          type="button"
          onClick={() => setAdding(true)}
          className="rounded-md bg-blue-600 px-2 py-1 text-xs font-medium text-white hover:bg-blue-700"
        >
          + 添加
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-2">
        {users.length === 0 ? (
          <div className="px-3 py-6 text-center text-xs text-zinc-400">该角色暂无成员</div>
        ) : (
          users.map((u) => {
            // admin 本地账号是系统兜底账号，从 admin 角色解绑会导致整个系统失去管理员入口，
            // 前端隐藏移除按钮、后端兜底拒绝（防御性双层）
            const isProtected = role.code === "admin" && u.username === "admin";
            return (
            <div
              key={u.user_id}
              className="group flex items-center justify-between rounded-lg px-3 py-2 hover:bg-zinc-50 dark:hover:bg-zinc-800"
            >
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium">
                  {u.username}
                  {isProtected && (
                    <span className="ml-1 rounded bg-zinc-200 px-1 text-[10px] text-zinc-600 dark:bg-zinc-700 dark:text-zinc-300">
                      内置
                    </span>
                  )}
                </div>
                <div className="truncate text-xs text-zinc-400">
                  {u.lark_open_id || "本地账号"}
                </div>
              </div>
              {!isProtected && (
              <button
                type="button"
                onClick={() => removeUser(u.user_id, u.username)}
                className="hidden text-xs text-red-600 hover:underline group-hover:inline"
              >
                移除
              </button>
              )}
            </div>
            );
          })
        )}
      </div>

      {adding && (
        <AddUserDialog
          roleID={role.id}
          roleName={role.name}
          onClose={() => setAdding(false)}
          onAdded={() => {
            setAdding(false);
            reload();
            onChanged();
          }}
        />
      )}
    </>
  );
}

// 添加用户弹窗：搜索飞书已同步用户 → 选中 → 调后端 POST /roles/:id/users
// 后端按 lark_open_id 匹配 PG users，未在白名单的用户加入失败（后端会忽略，前端不强校验）
function AddUserDialog({
  roleID,
  roleName,
  onClose,
  onAdded,
}: {
  roleID: string;
  roleName: string;
  onClose: () => void;
  onAdded: () => void;
}) {
  const [q, setQ] = useState("");
  const [list, setList] = useState<SyncedUser[]>([]);
  const [picked, setPicked] = useState<Map<string, SyncedUser>>(new Map());
  const [submitting, setSubmitting] = useState(false);

  // 输入防抖搜索飞书用户
  useEffect(() => {
    const t = setTimeout(async () => {
      if (!q.trim()) {
        setList([]);
        return;
      }
      const d = await fetch(
        `/api/proxy/contacts/search?q=${encodeURIComponent(q.trim())}&limit=12`,
      ).then((r) => r.json());
      setList(d.users ?? []);
    }, 250);
    return () => clearTimeout(t);
  }, [q]);

  const togglePick = (u: SyncedUser) => {
    setPicked((prev) => {
      const next = new Map(prev);
      if (next.has(u.id)) next.delete(u.id);
      else next.set(u.id, u);
      return next;
    });
  };

  // 提交：把 picked 用户的 PG user_id 传给后端
  // 但前端目前只有 Neo4j User 节点的 id（不是 PG users.id），需要后端按 lark_open_id 解析
  // 因此 POST 参数改成 lark_open_ids，由后端 join 转换为 PG user_id
  // —— 但当前 handler 设计是 user_ids，需要传 PG users.id；这里先做最直接版：
  // 通过 lark_open_id 调一个解析接口，或后端额外提供
  // 简化方案：本期假设 SyncedUser.id 与 PG users.id 一致（实际不一致），
  // 后端 RoleService.AddUsers 内部增强：若 user_ids 命中 lark_open_id 也接受
  const submit = async () => {
    if (picked.size === 0) return;
    setSubmitting(true);
    try {
      // 用 feishu_id (open_id) 列表传给后端，由后端解析为 PG user_id
      const openIDs = Array.from(picked.values()).map((u) => u.feishu_id);
      const res = await fetch(`/api/proxy/roles/${roleID}/users`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ lark_open_ids: openIDs }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        alert(body?.msg ?? "添加失败");
        return;
      }
      onAdded();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-lg rounded-xl bg-white p-5 shadow-xl dark:bg-zinc-900">
        <h3 className="mb-3 text-base font-semibold">
          添加成员到「{roleName}」
        </h3>
        <input
          type="search"
          autoFocus
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="按姓名 / 邮箱搜索飞书用户..."
          className="mb-3 w-full rounded-md border border-zinc-300 px-3 py-1.5 text-sm dark:border-zinc-700 dark:bg-zinc-900"
        />
        <div className="max-h-72 overflow-y-auto rounded-md border border-zinc-200 dark:border-zinc-800">
          {list.length === 0 ? (
            <div className="px-3 py-6 text-center text-xs text-zinc-400">
              {q.trim() ? "无匹配用户" : "输入关键字搜索"}
            </div>
          ) : (
            list.map((u) => {
              const isPicked = picked.has(u.id);
              return (
                <label
                  key={u.id}
                  className="flex cursor-pointer items-center gap-2 border-b border-zinc-100 px-3 py-2 hover:bg-zinc-50 dark:border-zinc-800 dark:hover:bg-zinc-800"
                >
                  <input
                    type="checkbox"
                    checked={isPicked}
                    onChange={() => togglePick(u)}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium">{u.name}</div>
                    <div className="truncate text-xs text-zinc-400">
                      {u.email || u.feishu_id}
                    </div>
                  </div>
                </label>
              );
            })
          )}
        </div>
        <div className="mt-3 text-xs text-zinc-500">已选 {picked.size} 人</div>
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
            disabled={picked.size === 0 || submitting}
            onClick={submit}
            className="rounded-md bg-blue-600 px-3 py-1.5 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {submitting ? "添加中..." : "添加"}
          </button>
        </div>
      </div>
    </div>
  );
}
