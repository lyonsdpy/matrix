import Link from "next/link";
import { LogoutButton } from "./logout-button";

// 登录后的主框架：左侧导航 + 顶栏。所有 (dashboard) 下的页面共享此布局。
export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen bg-zinc-50 text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100">
      {/* 侧边栏 */}
      <aside className="flex w-56 shrink-0 flex-col border-r border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div className="flex h-14 items-center px-5 text-lg font-semibold tracking-tight">
          Matrix
        </div>
        <nav className="flex-1 space-y-1 px-3 py-2">
          <NavItem href="/contacts" label="通讯录管理" />
          <NavItem href="/endpoints" label="终端管理" />
        </nav>
      </aside>

      {/* 主区域 */}
      <div className="flex flex-1 flex-col">
        <header className="flex h-14 items-center justify-end border-b border-zinc-200 bg-white px-6 dark:border-zinc-800 dark:bg-zinc-900">
          <LogoutButton />
        </header>
        <main className="flex-1 p-6">{children}</main>
      </div>
    </div>
  );
}

function NavItem({ href, label }: { href: string; label: string }) {
  return (
    <Link
      href={href}
      className="block rounded-lg px-3 py-2 text-sm font-medium text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-800"
    >
      {label}
    </Link>
  );
}
