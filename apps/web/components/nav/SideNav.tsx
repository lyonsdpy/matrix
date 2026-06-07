"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { findActiveSection, isLeafActive, filterNavByPermissions } from "./nav-config";
import { useMe } from "@/lib/use-me";

// 侧边二级菜单：根据当前激活的一级菜单展开其子项。
// 加载权限后按 useMe().permissions 过滤，无权限的子项不渲染。
// 无子项的一级（如"云网络管理"）显示占位文案，保持侧栏宽度稳定避免布局抖动。
export function SideNav() {
  const pathname = usePathname();
  const { permissions, loading } = useMe();

  // 权限拉取期间先按完整 NAV 渲染（避免一级菜单切换时侧栏抖动）；
  // 拉到后再过滤掉无权限的子项
  const sections = loading ? null : filterNavByPermissions(permissions);
  const allSection = findActiveSection(pathname);
  // 找到过滤后的对应 section；若已被过滤掉则回退用 allSection（避免空 section 报错）
  const section = sections?.find((s) => s.label === allSection.label) ?? allSection;

  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-zinc-200 px-4 py-3 text-xs font-semibold uppercase tracking-wide text-zinc-400 dark:border-zinc-800">
        {section.label}
      </div>
      <nav className="flex-1 space-y-1 px-3 py-3">
        {section.children.length === 0 ? (
          <p className="px-3 py-6 text-center text-xs text-zinc-400">暂无子项</p>
        ) : (
          section.children.map((c) => {
            const active = isLeafActive(pathname, c.href);
            return (
              <Link
                key={c.href}
                href={c.href}
                className={
                  active
                    ? "block rounded-lg bg-zinc-100 px-3 py-2 text-sm font-medium text-zinc-900 dark:bg-zinc-800 dark:text-zinc-100"
                    : "block rounded-lg px-3 py-2 text-sm font-medium text-zinc-600 transition-colors hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
                }
              >
                {c.label}
              </Link>
            );
          })
        )}
      </nav>
    </div>
  );
}
