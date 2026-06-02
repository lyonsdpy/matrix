"use client";

import { useEffect } from "react";
import type { EntityRef, EntityType } from "@/lib/types";
import { UserDetail } from "./UserDetail";
import { DepartmentDetail } from "./DepartmentDetail";

// 实体详情组件签名：未来加 Device/Group 等新类型，只需新建组件并注册到 registry 即可。
export type EntityDetailProps = {
  id: string;
  onSelectEntity: (e: EntityRef) => void;
};

export type EntityRegistryEntry = {
  title: string;
  Component: React.ComponentType<EntityDetailProps>;
};

// 默认 registry：user/department。其他页面（如终端管理）通过 extraRegistry prop 注入更多类型，
// 避免本文件反向 import 其他模块的详情组件造成循环依赖。
const defaultRegistry: Partial<Record<EntityType, EntityRegistryEntry>> = {
  user: { title: "用户详情", Component: UserDetail },
  department: { title: "部门详情", Component: DepartmentDetail },
};

export function EntityDrawer({
  entity,
  onClose,
  onSelectEntity,
  extraRegistry,
}: {
  entity: EntityRef | null;
  onClose: () => void;
  onSelectEntity: (e: EntityRef) => void;
  extraRegistry?: Partial<Record<EntityType, EntityRegistryEntry>>;
}) {
  // ESC 关闭抽屉
  useEffect(() => {
    if (!entity) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [entity, onClose]);

  if (!entity) return null;
  const merged: Partial<Record<EntityType, EntityRegistryEntry>> = {
    ...defaultRegistry,
    ...(extraRegistry ?? {}),
  };
  const entry = merged[entity.type];
  const Component = entry?.Component;

  return (
    <div className="fixed inset-0 z-50">
      {/* 半透明遮罩，点击关闭 */}
      <div className="absolute inset-0 bg-black/30 transition-opacity" onClick={onClose} />
      {/* 右侧抽屉 */}
      <div className="absolute right-0 top-0 flex h-full w-full max-w-xl flex-col bg-white shadow-2xl dark:bg-zinc-900">
        <div className="flex items-center justify-between border-b border-zinc-200 px-5 py-3 dark:border-zinc-800">
          <span className="text-sm font-semibold">{entry?.title ?? "详情"}</span>
          <button
            type="button"
            onClick={onClose}
            className="rounded p-1 text-zinc-500 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
            aria-label="关闭"
          >
            ✕
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-5">
          {Component ? (
            <Component key={`${entity.type}:${entity.id}`} id={entity.id} onSelectEntity={onSelectEntity} />
          ) : (
            <div className="text-sm text-zinc-400">未注册的实体类型：{entity.type}</div>
          )}
        </div>
      </div>
    </div>
  );
}
