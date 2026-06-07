import { RolesClient } from "@/components/roles/RolesClient";

// 角色权限管理：三栏布局（角色列表 / 权限勾选树 / 该角色用户）。
// 主交互在 RolesClient（client component），本页只负责标题与挂载。
export default function RolesPage() {
  return (
    <div className="flex h-full flex-col gap-4">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold">角色权限管理</h1>
      </div>
      <RolesClient />
    </div>
  );
}
