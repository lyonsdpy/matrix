"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";
import { findActiveSection } from "./nav-config";

// 浏览器 title 跟随当前一级菜单变化。
// SSR 阶段 title 由 root layout 的 metadata 决定（=Matrix）；
// 客户端水合后此组件按 pathname 同步 document.title，
// 进入 dashboard 子页面 → "Matrix · <一级菜单名>"。
// 离开 dashboard 时不会 unmount，需要在登录页另设 title（其 page.tsx 单独处理）。
export function DocumentTitle() {
  const pathname = usePathname();
  useEffect(() => {
    const section = findActiveSection(pathname);
    // section 总能找到（findActiveSection 兜底返回第一个一级），所以一定带后缀
    document.title = `Matrix · ${section.label}`;
  }, [pathname]);
  return null;
}
