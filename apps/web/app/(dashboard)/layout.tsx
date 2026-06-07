import { LogoutButton } from "./logout-button";
import { TopNav } from "@/components/nav/TopNav";
import { SideNav } from "@/components/nav/SideNav";
import { DocumentTitle } from "@/components/nav/DocumentTitle";

// 登录后主框架：顶部 header（Logo + 一级菜单 + 登出） + 下方左侧二级菜单 + 主区域。
// 外层 h-screen + overflow-hidden 保证整页贴合视口，子页面通过 flex 链路自然分配高度。
// 任何子页面如需局部滚动，在自己的容器上加 overflow-y-auto（参见通讯录左侧部门树）。
export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex h-screen flex-col overflow-hidden bg-zinc-50 text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100">
      {/* 浏览器 title 跟随一级菜单：进入 dashboard 后 "Matrix · 设备管理" 等 */}
      <DocumentTitle />
      {/* 顶部 header：横跨全宽。Logo 居左、一级菜单紧贴 Logo、登出居右。
          shrink-0 防止下方主区域过高时挤压 header。 */}
      <header className="flex h-14 shrink-0 items-center gap-6 border-b border-zinc-200 bg-white px-6 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="text-lg font-semibold tracking-tight">Matrix</div>
        <TopNav />
        <div className="ml-auto">
          <LogoutButton />
        </div>
      </header>

      {/* 主行：左侧栏 + 主区域；min-h-0 让本行能在 flex 列里正确收缩 */}
      <div className="flex min-h-0 flex-1">
        {/* 侧边栏：宽度恒定，渲染当前一级菜单的子项 */}
        <aside className="w-56 shrink-0 border-r border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          <SideNav />
        </aside>

        {/* 主区域：min-w-0 + min-h-0 配合 flex-1，防止子内容（长表格、长文本）撑破布局 */}
        <main className="min-h-0 min-w-0 flex-1 overflow-hidden p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
