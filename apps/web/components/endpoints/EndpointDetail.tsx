"use client";

import { useEffect, useState } from "react";
import type { EntityRef, SyncedEndpoint, SyncedUser } from "@/lib/types";

// 飞书原始编号 → 中文标签。来源：
// https://open.feishu.cn/document/security_and_compliance-v1/security_and_compliance-v2/device_record/list
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

// device_ownership：1=公司 2=个人（飞书文档未直接列出，按常见值约定）
const OWNERSHIP_LABEL: Record<string, string> = {
  "1": "公司设备",
  "2": "个人设备",
};

// device_status：1=可信 2=不可信（按飞书"可信状态"语义约定）
const TRUST_LABEL: Record<string, { text: string; klass: string }> = {
  "1": { text: "可信", klass: "bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300" },
  "2": { text: "不可信", klass: "bg-rose-50 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300" },
};

const CERT_LABEL: Record<string, string> = {
  "1": "已认证",
  "2": "未认证",
};

// EndpointDetail 终端详情卡片：基本信息 / 硬件标识 / 系统 / 合规 / MDM / 关联用户。
// onSelectEntity 让卡片内的用户跳转切换到 user 抽屉。
export function EndpointDetail({
  id,
  onSelectEntity,
}: {
  id: string;
  onSelectEntity: (e: EntityRef) => void;
}) {
  const [data, setData] = useState<SyncedEndpoint | null | "loading" | "error">("loading");

  useEffect(() => {
    setData("loading");
    fetch(`/api/proxy/endpoints/${encodeURIComponent(id)}`)
      .then(async (r) => {
        if (r.status === 404) return null;
        if (!r.ok) throw new Error(String(r.status));
        return (await r.json()) as SyncedEndpoint;
      })
      .then((d) => setData(d))
      .catch(() => setData("error"));
  }, [id]);

  if (data === "loading") return <div className="text-sm text-zinc-400">加载终端...</div>;
  if (data === "error") return <div className="text-sm text-red-500">加载失败</div>;
  if (data === null) return <div className="text-sm text-zinc-400">终端不存在</div>;

  const typeText = TYPE_LABEL[data.type] ?? data.type;
  const platformText = PLATFORM_LABEL[data.platform_code];
  const ownership = OWNERSHIP_LABEL[data.ownership];
  const trust = TRUST_LABEL[data.trust_level];
  const cert = CERT_LABEL[data.certification];

  return (
    <div className="space-y-6">
      {/* 标题区 */}
      <div>
        <div className="flex items-start gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-slate-400 to-slate-600 text-lg font-semibold text-white">
            {emojiForType(data.type)}
          </div>
          <div className="min-w-0">
            <div className="break-all text-base font-semibold">{data.name || "—"}</div>
            <div className="mt-0.5 text-xs text-zinc-500">
              {typeText}
              {platformText && <span className="ml-1">· {platformText}</span>}
              {data.version && <span className="ml-1">· v{data.version}</span>}
            </div>
            <div className="mt-2 flex flex-wrap items-center gap-1.5">
              {trust && (
                <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${trust.klass}`}>
                  {trust.text}
                </span>
              )}
              {ownership && (
                <span className="rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">
                  {ownership}
                </span>
              )}
              {cert && (
                <span className="rounded-full bg-zinc-100 px-2 py-0.5 text-xs text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400">
                  {cert}
                </span>
              )}
              {data.is_managed && (
                <span className="rounded-full bg-violet-50 px-2 py-0.5 text-xs font-medium text-violet-700 dark:bg-violet-900/30 dark:text-violet-300">
                  MDM 受管
                </span>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* 关联用户 */}
      <Section title="关联用户">
        <div className="space-y-2">
          <UserRow label="当前登录" user={data.current_user} onSelect={onSelectEntity} />
          <UserRow label="最近登录" user={data.latest_user} onSelect={onSelectEntity} />
        </div>
      </Section>

      {/* 硬件标识 */}
      <Section title="硬件标识">
        <KvGrid
          rows={[
            ["设备型号", data.model],
            ["生产序列号", mono(data.serial_number)],
            ["硬盘序列号", mono(data.disk_serial_number)],
            ["主板 UUID", mono(data.board_uuid)],
            ["MAC 地址", mono(data.mac_address)],
          ]}
        />
      </Section>

      {/* MDM */}
      {(data.mdm_device_id || data.mdm_provider) && (
        <Section title="MDM 管理">
          <KvGrid
            rows={[
              ["MDM 厂商", data.mdm_provider],
              ["MDM 设备 ID", mono(data.mdm_device_id)],
            ]}
          />
        </Section>
      )}

      {/* 飞书原始标识 */}
      <Section title="飞书标识">
        <KvGrid
          rows={[
            ["device_record_id", mono(data.feishu_device_id)],
            ["节点 id", mono(data.id)],
          ]}
        />
      </Section>
    </div>
  );
}

function UserRow({
  label,
  user,
  onSelect,
}: {
  label: string;
  user?: SyncedUser;
  onSelect: (e: EntityRef) => void;
}) {
  if (!user) {
    return (
      <div className="flex items-center justify-between rounded-lg border border-dashed border-zinc-300 px-3 py-2 text-sm text-zinc-400 dark:border-zinc-700">
        <span>{label}</span>
        <span>—</span>
      </div>
    );
  }
  return (
    <button
      type="button"
      onClick={() => onSelect({ type: "user", id: user.feishu_id })}
      className="flex w-full items-center justify-between rounded-lg border border-zinc-200 px-3 py-2 text-left hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800"
    >
      <div>
        <div className="text-xs text-zinc-500">{label}</div>
        <div className="text-sm font-medium text-blue-600 hover:underline dark:text-blue-400">
          {user.name || "—"}
        </div>
        {user.email && (
          <div className="text-xs text-zinc-500">{user.email}</div>
        )}
      </div>
      <span className="text-xs text-zinc-400">查看 →</span>
    </button>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500">
        {title}
      </div>
      {children}
    </div>
  );
}

function KvGrid({ rows }: { rows: Array<[string, React.ReactNode]> }) {
  return (
    <div className="grid grid-cols-[100px_1fr] gap-y-2 text-sm">
      {rows.map(([k, v]) => (
        <KvRow key={k} k={k} v={v} />
      ))}
    </div>
  );
}

function KvRow({ k, v }: { k: string; v: React.ReactNode }) {
  return (
    <>
      <span className="text-zinc-500">{k}</span>
      <span className="break-all">{v || <span className="text-zinc-400">—</span>}</span>
    </>
  );
}

function mono(text: string) {
  if (!text) return undefined;
  return <span className="font-mono text-xs text-zinc-600 dark:text-zinc-400">{text}</span>;
}

function emojiForType(t: string): string {
  switch (t) {
    case "PC":
      return "🖥";
    case "LAPTOP":
      return "💻";
    case "PHONE":
      return "📱";
    case "TABLET":
      return "📱";
    case "PRINTER":
      return "🖨";
    case "TV":
      return "📺";
    case "ATTENDANCE":
      return "🕘";
    default:
      return "🔌";
  }
}
