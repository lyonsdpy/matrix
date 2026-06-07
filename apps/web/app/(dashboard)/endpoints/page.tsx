import { EndpointsClient } from "@/components/endpoints/EndpointsClient";

// 终端管理：单列表 + 双搜索（设备名/序列号 + 关联用户）+ 游标分页。
// 主交互在 client component，本页只负责标题与挂载。
export default function EndpointsPage() {
  // h-full + flex-col：占满 layout 留给 main 的高度，让 EndpointsClient 用 flex-1 接管剩余空间。
  return (
    <div className="flex h-full flex-col gap-4">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold">终端管理</h1>
      </div>
      <EndpointsClient />
    </div>
  );
}
