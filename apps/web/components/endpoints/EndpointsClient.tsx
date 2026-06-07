"use client";

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import type { EndpointList, EntityRef, SyncedEndpoint, SyncedUser } from "@/lib/types";
import { SyncEndpointsButton } from "./SyncEndpointsButton";
import { EntityDrawer } from "@/components/contacts/EntityDrawer";
import { EndpointDetail } from "./EndpointDetail";

// 注入到 EntityDrawer 的 endpoint registry：drawer 本身只默认注册 user/department，
// endpoint 详情组件由本文件按需注入，避免 EntityDrawer 反向 import 形成模块循环。
const endpointRegistry = {
  endpoint: { title: "终端详情", Component: EndpointDetail },
};

// 物理形态：与后端 EndpointType 枚举对齐（来自飞书 device_terminal_type）。
const TYPE_LABEL: Record<string, string> = {
  DESKTOP: "桌面端",
  MOBILE: "移动端",
  UNKNOWN: "未知",
};

// 操作系统：与后端 EndpointOS 枚举对齐。
const OS_LABEL: Record<string, string> = {
  WINDOWS: "Windows",
  MACOS: "macOS",
  LINUX: "Linux",
  IOS: "iOS",
  ANDROID: "Android",
  HARMONYOS: "HarmonyOS",
  OTHER: "其他",
};

// 下拉筛选项：value 为空字符串表示不筛。顺序为常用度优先。
const TYPE_FILTER_OPTIONS: Array<{ value: string; label: string }> = [
  { value: "", label: "全部类型" },
  { value: "DESKTOP", label: "桌面端" },
  { value: "MOBILE", label: "移动端" },
  { value: "UNKNOWN", label: "未知" },
];

const OS_FILTER_OPTIONS: Array<{ value: string; label: string }> = [
  { value: "", label: "全部系统" },
  { value: "WINDOWS", label: "Windows" },
  { value: "MACOS", label: "macOS" },
  { value: "HARMONYOS", label: "HarmonyOS" },
  { value: "ANDROID", label: "Android" },
  { value: "IOS", label: "iOS" },
  { value: "LINUX", label: "Linux" },
  { value: "OTHER", label: "其他" },
];

const STATUS_LABEL: Record<string, { text: string; klass: string }> = {
  ACTIVE: { text: "在用", klass: "bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300" },
  INACTIVE: { text: "未启用", klass: "bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400" },
  LOST: { text: "丢失", klass: "bg-rose-50 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300" },
  RETIRED: { text: "退役", klass: "bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400" },
};

// 行高常量：与表格 cell 的 py-2.5 + text-sm + border-b 实际渲染高度对齐。
// 用于按容器可用高度反推"恰好填满一屏"的 pageSize。
const ROW_HEIGHT = 41;
const MIN_PAGE_SIZE = 5;
const MAX_PAGE_SIZE = 200;

// 终端管理主交互：双搜索（设备 q + 用户 user）+ 类型/系统筛选 + 自适应分页器。
// pageSize 跟随表格容器高度动态计算（ResizeObserver），保证整页不出现滚动条。
export function EndpointsClient() {
  const [q, setQ] = useState("");
  const [userQ, setUserQ] = useState("");
  const [typeFilter, setTypeFilter] = useState("");
  const [osFilter, setOsFilter] = useState("");
  const [items, setItems] = useState<SyncedEndpoint[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>("");
  const [drawerEntity, setDrawerEntity] = useState<EntityRef | null>(null);

  // 表体可用高度测量：tbody 的祖先 div 注入 ref，ResizeObserver 触发 pageSize 重算。
  const tableBodyRef = useRef<HTMLDivElement | null>(null);
  // 防过期：每次 q/userQ/page/... 变化重置一个 token，回调里检查 token 是否还是最新。
  const reqToken = useRef(0);

  const totalPages = Math.max(1, Math.ceil(total / Math.max(1, pageSize)));

  // pageSize 自适应：根据 tbody 容器实际可用高度反推可容纳行数。
  // 用 useLayoutEffect 保证首次渲染就有合理值，避免"先按默认 20 请求一次再改 size 又请求一次"。
  useLayoutEffect(() => {
    const el = tableBodyRef.current;
    if (!el) return;
    const recompute = () => {
      const h = el.clientHeight;
      if (h <= 0) return;
      const next = Math.max(MIN_PAGE_SIZE, Math.min(MAX_PAGE_SIZE, Math.floor(h / ROW_HEIGHT)));
      setPageSize((prev) => (prev === next ? prev : next));
    };
    recompute();
    const ro = new ResizeObserver(recompute);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  // pageSize 变化时回到第 1 页（保持当前筛选条件），避免 page > totalPages。
  useEffect(() => {
    setPage(1);
  }, [pageSize, q, userQ, typeFilter, osFilter]);

  // 拉取当页：q/userQ 走 300ms 防抖；其余（page、pageSize、筛选）立即触发。
  useEffect(() => {
    const my = ++reqToken.current;
    const timer = setTimeout(
      () => {
        void fetchPage(my);
      },
      // 文本搜索防抖；点击翻页/改 size/换 filter 不需要等。
      300,
    );
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q, userQ, typeFilter, osFilter, page, pageSize]);

  // 同步完成后强制刷新当前页（不重置筛选/页码）
  const refetch = useCallback(() => {
    const my = ++reqToken.current;
    void fetchPage(my);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function fetchPage(token: number) {
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams();
      if (q) params.set("q", q);
      if (userQ) params.set("user", userQ);
      if (typeFilter) params.set("type", typeFilter);
      if (osFilter) params.set("os", osFilter);
      params.set("page", String(page));
      params.set("page_size", String(pageSize));
      const res = await fetch(`/api/proxy/endpoints?${params.toString()}`);
      if (token !== reqToken.current) return;
      if (!res.ok) {
        setError(`查询失败 (${res.status})`);
        return;
      }
      const data = (await res.json()) as EndpointList;
      setItems(data.endpoints ?? []);
      setTotal(data.total ?? 0);
    } catch (e) {
      if (token !== reqToken.current) return;
      setError((e as Error).message || "网络错误");
    } finally {
      if (token === reqToken.current) setLoading(false);
    }
  }

  return (
    // 父级 dashboard layout 已限定 h-screen，本组件用 flex-1 占满 main 留下的剩余高度，
    // min-h-0 让自身可以在 flex 列里正确收缩；内部用 flex-col 把"筛选栏 / 列表 / 分页器"按内容高度分配。
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      {/* 同步按钮：放页面顶部，便于"先同步再查"的操作链路 */}
      <div className="flex items-center justify-end">
        <SyncEndpointsButton onDone={refetch} />
      </div>

      {/* 搜索 + 筛选栏：两个文本搜索 + 两个下拉精确筛选 */}
      <div className="flex flex-wrap items-center gap-3 rounded-xl border border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-900">
        <SearchField
          label="设备"
          placeholder="设备名 / 序列号"
          value={q}
          onChange={setQ}
        />
        <SearchField
          label="关联用户"
          placeholder="姓名 / 邮箱（当前或最近登录）"
          value={userQ}
          onChange={setUserQ}
        />
        <SelectField
          label="类型"
          value={typeFilter}
          onChange={setTypeFilter}
          options={TYPE_FILTER_OPTIONS}
        />
        <SelectField
          label="系统"
          value={osFilter}
          onChange={setOsFilter}
          options={OS_FILTER_OPTIONS}
        />
        {loading && <span className="text-xs text-zinc-400">加载中...</span>}
        {error && <span className="text-xs text-rose-500">{error}</span>}
      </div>

      {/* 列表卡片：flex-1 撑满剩余空间；表头固定、表体 flex-1 + overflow-hidden 用于测量；分页器贴底。 */}
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div className="overflow-hidden">
          {/* table-fixed + 共用 colgroup：表头/表体两张独立 table 必须按同一套列宽渲染才能对齐。 */}
          <table className="w-full table-fixed text-sm">
            <ColGroup />
            <thead className="border-b border-zinc-200 bg-zinc-50 text-left text-xs uppercase tracking-wide text-zinc-500 dark:border-zinc-800 dark:bg-zinc-800/50">
              <tr>
                <th className="px-4 py-2.5 font-medium">设备名</th>
                <th className="px-4 py-2.5 font-medium">类型</th>
                <th className="px-4 py-2.5 font-medium">系统</th>
                <th className="px-4 py-2.5 font-medium">序列号</th>
                <th className="px-4 py-2.5 font-medium">最近登录用户</th>
                <th className="px-4 py-2.5 font-medium">状态</th>
              </tr>
            </thead>
          </table>
        </div>
        {/* 表体容器：撑满剩余高度，pageSize 按其 clientHeight 反推。
            不开启内部滚动 —— 行数已对齐高度，正常情况下不会溢出；
            极端 zoom/字体放大时 overflow-hidden 把多出的最后一行隐藏，比出现 scrollbar 干扰更可控。 */}
        <div ref={tableBodyRef} className="min-h-0 flex-1 overflow-hidden">
          <table className="w-full table-fixed text-sm">
            <ColGroup />
            <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800">
              {items.length === 0 && !loading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-10 text-center text-sm text-zinc-400">
                    无匹配设备
                  </td>
                </tr>
              ) : (
                items.map((ep) => (
                  <tr key={ep.id} className="hover:bg-zinc-50 dark:hover:bg-zinc-800/50">
                    <td className="px-4 py-2.5 font-medium">
                      <button
                        type="button"
                        onClick={() => setDrawerEntity({ type: "endpoint", id: ep.id })}
                        className="block w-full truncate text-left text-blue-600 hover:underline dark:text-blue-400"
                        title={ep.name || ""}
                      >
                        {ep.name || "—"}
                      </button>
                    </td>
                    <td className="truncate px-4 py-2.5 text-zinc-600 dark:text-zinc-400">
                      {TYPE_LABEL[ep.type] ?? ep.type ?? "—"}
                    </td>
                    <td className="truncate px-4 py-2.5 text-zinc-600 dark:text-zinc-400">
                      {OS_LABEL[ep.os] ?? ep.os ?? "—"}
                      {ep.version && (
                        <span className="ml-1 text-xs text-zinc-400">· v{ep.version}</span>
                      )}
                    </td>
                    <td
                      className="truncate px-4 py-2.5 font-mono text-xs text-zinc-600 dark:text-zinc-400"
                      title={ep.serial_number || ""}
                    >
                      {ep.serial_number || "—"}
                    </td>
                    <td className="px-4 py-2.5">
                      <UserCell user={ep.latest_user} onSelect={setDrawerEntity} />
                    </td>
                    <td className="px-4 py-2.5">
                      <StatusChip status={ep.status} />
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* 分页器：上一页 / 页码列表 / 下一页 + 总数。
            页码列表用 "1 ... k-1 k k+1 ... N" 折叠策略，最多渲染 7 个数字按钮。 */}
        <Pagination
          page={page}
          totalPages={totalPages}
          total={total}
          pageSize={pageSize}
          onChange={setPage}
        />
      </div>

      {/* 详情抽屉：endpoint / user / department 都路由到此 */}
      <EntityDrawer
        entity={drawerEntity}
        onClose={() => setDrawerEntity(null)}
        onSelectEntity={setDrawerEntity}
        extraRegistry={endpointRegistry}
      />
    </div>
  );
}

// 列宽定义：表头/表体两张 table 共用同一份 colgroup，是 fixed layout 下唯一可靠的对齐手段。
// 百分比之和 100%，按"信息密度 / 可读性"分配；状态/类型这种短词列窄一些。
// 6 列：设备名 / 类型 / 系统 / 序列号 / 最近登录用户 / 状态。
function ColGroup() {
  return (
    <colgroup>
      <col style={{ width: "24%" }} />
      <col style={{ width: "10%" }} />
      <col style={{ width: "14%" }} />
      <col style={{ width: "20%" }} />
      <col style={{ width: "22%" }} />
      <col style={{ width: "10%" }} />
    </colgroup>
  );
}

function SearchField({
  label,
  placeholder,
  value,
  onChange,
}: {
  label: string;
  placeholder: string;
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <label className="flex flex-1 min-w-[240px] items-center gap-2 text-sm">
      <span className="shrink-0 text-zinc-500">{label}</span>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="flex-1 rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm placeholder:text-zinc-400 focus:border-zinc-400 focus:outline-none dark:border-zinc-700 dark:bg-zinc-800 dark:placeholder:text-zinc-500"
      />
    </label>
  );
}

function SelectField({
  label,
  value,
  onChange,
  options,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  options: Array<{ value: string; label: string }>;
}) {
  return (
    <label className="flex items-center gap-2 text-sm">
      <span className="shrink-0 text-zinc-500">{label}</span>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="rounded-md border border-zinc-300 bg-white px-2 py-1.5 text-sm focus:border-zinc-400 focus:outline-none dark:border-zinc-700 dark:bg-zinc-800"
      >
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </label>
  );
}

function UserCell({
  user,
  onSelect,
}: {
  user?: SyncedUser;
  onSelect: (e: EntityRef) => void;
}) {
  if (!user) {
    return <span className="text-xs text-zinc-400">—</span>;
  }
  return (
    <button
      type="button"
      onClick={() => onSelect({ type: "user", id: user.feishu_id })}
      className="flex w-full min-w-0 flex-col text-left"
      title={user.email || user.name || ""}
    >
      <span className="truncate font-medium text-blue-600 hover:underline dark:text-blue-400">
        {user.name || "—"}
      </span>
      {user.email && (
        <span className="truncate text-xs text-zinc-500">{user.email}</span>
      )}
    </button>
  );
}

function StatusChip({ status }: { status: string }) {
  const conf = STATUS_LABEL[status] ?? {
    text: status || "—",
    klass: "bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400",
  };
  return (
    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${conf.klass}`}>
      {conf.text}
    </span>
  );
}

// 分页器：紧凑布局，左侧总数信息、右侧分页控件。
// 页码折叠规则：始终显示首尾，中间显示 [current-1, current, current+1]，省略段用 "…"。
function Pagination({
  page,
  totalPages,
  total,
  pageSize,
  onChange,
}: {
  page: number;
  totalPages: number;
  total: number;
  pageSize: number;
  onChange: (p: number) => void;
}) {
  const pages = useMemo(() => buildPageList(page, totalPages), [page, totalPages]);
  const canPrev = page > 1;
  const canNext = page < totalPages;
  return (
    <div className="flex items-center justify-between gap-3 border-t border-zinc-200 px-4 py-2 text-xs text-zinc-500 dark:border-zinc-800">
      <span>
        共 <span className="font-medium text-zinc-700 dark:text-zinc-300">{total}</span> 条 · 每页{" "}
        <span className="font-medium text-zinc-700 dark:text-zinc-300">{pageSize}</span> 条
      </span>
      <div className="flex items-center gap-1">
        <PageBtn disabled={!canPrev} onClick={() => onChange(page - 1)}>
          上一页
        </PageBtn>
        {pages.map((p, i) =>
          p === "..." ? (
            <span key={`gap-${i}`} className="px-1 text-zinc-400">
              …
            </span>
          ) : (
            <PageBtn
              key={p}
              active={p === page}
              onClick={() => onChange(p)}
            >
              {p}
            </PageBtn>
          ),
        )}
        <PageBtn disabled={!canNext} onClick={() => onChange(page + 1)}>
          下一页
        </PageBtn>
      </div>
    </div>
  );
}

function PageBtn({
  children,
  onClick,
  active,
  disabled,
}: {
  children: React.ReactNode;
  onClick?: () => void;
  active?: boolean;
  disabled?: boolean;
}) {
  // 三态样式：active = 蓝底白字 / disabled = 灰字不可点 / 默认 = 边框透明 hover 变浅灰底
  const base = "inline-flex min-w-[28px] items-center justify-center rounded-md px-2 py-1 text-xs transition-colors";
  const cls = active
    ? `${base} bg-blue-600 font-medium text-white`
    : disabled
      ? `${base} cursor-not-allowed text-zinc-300 dark:text-zinc-600`
      : `${base} text-zinc-600 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-800`;
  return (
    <button type="button" onClick={onClick} disabled={disabled} className={cls}>
      {children}
    </button>
  );
}

// 折叠页码列表：保持首尾页 + 当前页前后各 1 页 + 省略段。
// totalPages<=7 时全部列出，避免出现单个省略号反而占位的尴尬。
function buildPageList(current: number, totalPages: number): (number | "...")[] {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }
  const result: (number | "...")[] = [1];
  const start = Math.max(2, current - 1);
  const end = Math.min(totalPages - 1, current + 1);
  if (start > 2) result.push("...");
  for (let p = start; p <= end; p++) result.push(p);
  if (end < totalPages - 1) result.push("...");
  result.push(totalPages);
  return result;
}
