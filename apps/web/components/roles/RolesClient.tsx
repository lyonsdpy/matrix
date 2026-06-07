"use client";

import { useCallback, useEffect, useState } from "react";
import type { Permission, Role, RoleDetail } from "@/lib/types";
import { RoleList } from "./RoleList";
import { PermissionTree } from "./PermissionTree";
import { RoleUsers } from "./RoleUsers";

// 角色权限管理三栏布局：
//   左栏 220：角色列表（新建/编辑/删除）
//   中栏 flex-1：权限勾选树（按 module 分组，第一期只渲染 kind=page）
//   右栏 320：该角色下的用户（添加/移除）
//
// 选中角色后中右两栏同步刷新；admin 角色特殊渲染（中间件特判，UI 不显示权限树）
export function RolesClient() {
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [selectedRoleId, setSelectedRoleId] = useState<string>("");
  const [detail, setDetail] = useState<RoleDetail | null>(null);

  const reloadRoles = useCallback(async () => {
    const r = await fetch("/api/proxy/roles").then((r) => r.json());
    const items: Role[] = r.items ?? [];
    setRoles(items);
    // 默认选中第一个；若当前选中已被删则回退
    if (items.length === 0) {
      setSelectedRoleId("");
    } else if (!items.find((x) => x.id === selectedRoleId)) {
      setSelectedRoleId(items[0].id);
    }
  }, [selectedRoleId]);

  // 初次加载：角色列表 + 权限码全集（权限码相对静态，无需在选中变化时重拉）
  useEffect(() => {
    reloadRoles();
    fetch("/api/proxy/permissions")
      .then((r) => r.json())
      .then((d) => setPermissions(d.items ?? []));
  }, [reloadRoles]);

  // 选中角色变化 → 拉详情（含权限码集合）
  useEffect(() => {
    if (!selectedRoleId) {
      setDetail(null);
      return;
    }
    fetch(`/api/proxy/roles/${selectedRoleId}`)
      .then((r) => (r.ok ? r.json() : null))
      .then(setDetail);
  }, [selectedRoleId]);

  return (
    <div className="flex min-h-0 flex-1 gap-4">
      {/* 左栏：角色列表 */}
      <div className="flex w-60 shrink-0 flex-col overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <RoleList
          roles={roles}
          selectedId={selectedRoleId}
          onSelect={setSelectedRoleId}
          onChanged={reloadRoles}
        />
      </div>

      {/* 中栏：权限勾选树 */}
      <div className="flex flex-1 flex-col overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        {!detail ? (
          <EmptyMid />
        ) : (
          <PermissionTree
            role={detail}
            permissions={permissions}
            onSaved={() => {
              if (selectedRoleId) {
                fetch(`/api/proxy/roles/${selectedRoleId}`)
                  .then((r) => r.json())
                  .then(setDetail);
              }
            }}
          />
        )}
      </div>

      {/* 右栏：该角色下用户 */}
      <div className="flex w-80 shrink-0 flex-col overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        {!detail ? (
          <EmptyRight />
        ) : (
          <RoleUsers role={detail} onChanged={reloadRoles} />
        )}
      </div>
    </div>
  );
}

function EmptyMid() {
  return (
    <div className="flex flex-1 items-center justify-center text-sm text-zinc-400">
      从左侧选择角色查看/编辑权限
    </div>
  );
}

function EmptyRight() {
  return (
    <div className="flex flex-1 items-center justify-center text-sm text-zinc-400">
      角色下的成员
    </div>
  );
}
