import { EndpointsClient } from "@/components/endpoints/EndpointsClient";

// 终端管理：单列表 + 双搜索（设备名/序列号 + 关联用户）+ 游标分页。
// 主交互在 client component，本页只负责标题与挂载。
export default function EndpointsPage() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold">终端管理</h1>
      </div>
      <EndpointsClient />
    </div>
  );
}
