"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { NAV, findActiveSection, filterNavByPermissions } from "./nav-config";
import { useMe } from "@/lib/use-me";

// 顶部一级菜单：用 pathname 反查激活项，点击跳到该一级的默认入口（href = 第一个子项）。
// 按 useMe().permissions 过滤掉无权限的一级菜单；权限未加载完前按完整 NAV 渲染避免闪烁
export function TopNav() {
  const pathname = usePathname();
  const { permissions, loading } = useMe();
  const sections = loading ? NAV : filterNavByPermissions(permissions);
  const active = findActiveSection(pathname);
  return (
    <nav className="flex h-full items-center gap-1">
      {sections.map((s) => {
        const isActive = s.label === active.label;
        // 一级菜单的 href 必须指向当前可见的入口（首个可见 child 或自身）
        // 否则点击会跳到被过滤掉的子页，被后端 403 拦截
        const href = s.children.length > 0 ? s.children[0].href : s.href;
        return (
          <Link
            key={s.label}
            href={href}
            className={
              isActive
                ? "rounded-md bg-blue-50 px-3 py-1.5 text-sm font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
                : "rounded-md px-3 py-1.5 text-sm font-medium text-zinc-600 transition-colors hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
            }
          >
            {s.label}
          </Link>
        );
      })}
    </nav>
  );
}
