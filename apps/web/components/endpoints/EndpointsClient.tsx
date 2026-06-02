"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import type { EndpointList, EntityRef, SyncedEndpoint, SyncedUser } from "@/lib/types";
import { SyncEndpointsButton } from "./SyncEndpointsButton";
import { EntityDrawer } from "@/components/contacts/EntityDrawer";
import { EndpointDetail } from "./EndpointDetail";

// 注入到 EntityDrawer 的 endpoint registry：drawer 本身只默认注册 user/department，
// endpoint 详情组件由本文件按需注入，避免 EntityDrawer 反向 import 形成模块循环。
const endpointRegistry = {
  endpoint: { title: "终端详情", Component: EndpointDetail },
};

// 飞书 terminal_type 编号 → 中文标签（与后端 terminalTypeToEndpoint 同源）。
const PLATFORM_LABEL: Record<string, string> = {
  "1": "Windows",
  "2": "macOS",
  "3": "Linux",
  "4": "iOS",
  "5": "Android",
};

const TYPE_LABEL: Record<string, string> = {
  PC: "电脑",
  LAPTOP: "笔记本",
  PRINTER: "打印机",
  TV: "电视",
  ATTENDANCE: "考勤机",
  PHONE: "手机",
  TABLET: "平板",
  OTHER: "其他",
};

const STATUS_LABEL: Record<string, { text: string; klass: string }> = {
  ACTIVE: { text: "在用", klass: "bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300" },
  INACTIVE: { text: "未启用", klass: "bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400" },
  LOST: { text: "丢失", klass: "bg-rose-50 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300" },
  RETIRED: { text: "退役", klass: "bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400" },
};

const PAGE_SIZE = 30;

// 终端管理主交互：双搜索（设备 q + 用户 user）+ 列表 + 加载更多。
// 搜索 input 输入后 300ms 防抖触发重查；分页用游标 endCursor。
export function EndpointsClient() {
  const [q, setQ] = useState("");
  const [userQ, setUserQ] = useState("");
  const [items, setItems] = useState<SyncedEndpoint[]>([]);
  const [cursor, setCursor] = useState<string>("");
  const [hasNext, setHasNext] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>("");
  const [drawerEntity, setDrawerEntity] = useState<EntityRef | null>(null);

  // 防抖标识：每次 q/userQ 变化重置一个 token，回调里检查 token 是否还是最新
  const reqToken = useRef(0);

  useEffect(() => {
    const my = ++reqToken.current;
    const timer = setTimeout(() => {
      void fetchPage({ reset: true, token: my });
    }, 300);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q, userQ]);

  // 同步完成后强制重查（按当前 q/userQ），不动用户的搜索输入
  const refetch = useCallback(() => {
    const my = ++reqToken.current;
    void fetchPage({ reset: true, token: my });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function fetchPage(opts: { reset: boolean; token?: number }) {
    const my = opts.token ?? ++reqToken.current;
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams();
      if (q) params.set("q", q);
      if (userQ) params.set("user", userQ);
      if (!opts.reset && cursor) params.set("cursor", cursor);
      params.set("limit", String(PAGE_SIZE));
      const res = await fetch(`/api/proxy/endpoints?${params.toString()}`);
      // 防过期：发起后 q/userQ 又变化了，丢弃本次结果
      if (my !== reqToken.current) return;
      if (!res.ok) {
        setError(`查询失败 (${res.status})`);
        return;
      }
      const data = (await res.json()) as EndpointList;
      const list = data.endpoints ?? [];
      setItems(opts.reset ? list : [...items, ...list]);
      setCursor(data.end_cursor ?? "");
      setHasNext(!!data.has_next);
    } catch (e) {
      if (my !== reqToken.current) return;
      setError((e as Error).message || "网络错误");
    } finally {
      if (my === reqToken.current) setLoading(false);
    }
  }

  return (
    <div className="flex flex-col gap-3">
      {/* 同步按钮：放页面顶部，便于"先同步再查"的操作链路 */}
      <div className="flex items-center justify-end">
        <SyncEndpointsButton onDone={refetch} />
      </div>

      {/* 双搜索栏 */}
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
        {loading && <span className="text-xs text-zinc-400">加载中...</span>}
        {error && <span className="text-xs text-rose-500">{error}</span>}
      </div>

      {/* 列表 */}
      <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <table className="w-full text-sm">
          <thead className="border-b border-zinc-200 bg-zinc-50 text-left text-xs uppercase tracking-wide text-zinc-500 dark:border-zinc-800 dark:bg-zinc-800/50">
            <tr>
              <th className="px-4 py-2.5 font-medium">设备名</th>
              <th className="px-4 py-2.5 font-medium">类型</th>
              <th className="px-4 py-2.5 font-medium">序列号</th>
              <th className="px-4 py-2.5 font-medium">当前登录用户</th>
              <th className="px-4 py-2.5 font-medium">最近登录用户</th>
              <th className="px-4 py-2.5 font-medium">状态</th>
            </tr>
          </thead>
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
                      className="text-left text-blue-600 hover:underline dark:text-blue-400"
                    >
                      {ep.name || "—"}
                    </button>
                  </td>
                  <td className="px-4 py-2.5 text-zinc-600 dark:text-zinc-400">
                    {TYPE_LABEL[ep.type] ?? ep.type}
                    {ep.platform_code && PLATFORM_LABEL[ep.platform_code] && (
                      <span className="ml-1 text-xs text-zinc-400">
                        · {PLATFORM_LABEL[ep.platform_code]}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 font-mono text-xs text-zinc-600 dark:text-zinc-400">
                    {ep.serial_number || "—"}
                  </td>
                  <td className="px-4 py-2.5">
                    <UserCell user={ep.current_user} onSelect={setDrawerEntity} />
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

        {/* 加载更多 */}
        {hasNext && (
          <div className="border-t border-zinc-200 p-3 text-center dark:border-zinc-800">
            <button
              type="button"
              onClick={() => void fetchPage({ reset: false })}
              disabled={loading}
              className="rounded-lg border border-zinc-300 px-4 py-1.5 text-xs font-medium hover:bg-zinc-50 disabled:opacity-50 dark:border-zinc-700 dark:hover:bg-zinc-800"
            >
              {loading ? "加载中..." : "加载更多"}
            </button>
          </div>
        )}
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
      className="flex flex-col text-left"
    >
      <span className="font-medium text-blue-600 hover:underline dark:text-blue-400">
        {user.name || "—"}
      </span>
      {user.email && (
        <span className="text-xs text-zinc-500">{user.email}</span>
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
