"use client";

import { useEffect, useRef, useState } from "react";

// 同步进度快照，与 Go SyncService.Snapshot 输出保持一致（仅通讯录字段）。
type Counts = { created: number; updated: number; deleted: number; total: number };
type Phase =
  | "idle"
  | "fetch_departments"
  | "write_departments"
  | "fetch_users"
  | "write_users"
  | "finalizing"
  | "done"
  | "failed";

interface Progress {
  job_id: string;
  phase: Phase;
  started_at: string;
  finished_at?: string;
  duration_ms: number;
  done: number;
  total: number;
  departments: Counts;
  users: Counts;
  error?: string;
}

const PHASE_LABEL: Record<Phase, string> = {
  idle: "未运行",
  fetch_departments: "拉取部门",
  write_departments: "写入部门",
  fetch_users: "拉取用户",
  write_users: "写入用户",
  finalizing: "清理过期数据",
  done: "完成",
  failed: "失败",
};

const POLL_MS = 500;

// SyncContactsButton 触发飞书通讯录同步并以进度条 + 结果 chip 展示。
// 仅同步部门 + 用户；设备同步由"终端管理"页的独立按钮负责。
export function SyncContactsButton() {
  const [progress, setProgress] = useState<Progress | null>(null);
  const [triggering, setTriggering] = useState(false);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const running =
    progress !== null &&
    progress.phase !== "idle" &&
    progress.phase !== "done" &&
    progress.phase !== "failed";

  // 卸载清理轮询
  useEffect(() => {
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, []);

  // 进入页面时拍一次进度：可能有其他终端已在跑
  useEffect(() => {
    fetchProgress();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 进入 running 态时开始轮询；到终态自动停
  useEffect(() => {
    if (running && !timerRef.current) {
      timerRef.current = setInterval(fetchProgress, POLL_MS);
    }
    if (!running && timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
  }, [running]);

  async function fetchProgress() {
    try {
      const r = await fetch("/api/proxy/contacts/sync/progress", { cache: "no-store" });
      if (!r.ok) return;
      const p = (await r.json()) as Progress;
      setProgress(p);
    } catch {
      // 网络抖动忽略，下一轮再试
    }
  }

  async function startSync() {
    setTriggering(true);
    try {
      const r = await fetch("/api/proxy/contacts/sync/start", { method: "POST" });
      if (!r.ok) {
        const txt = await r.text();
        setProgress((prev) => ({
          ...(prev ?? blankProgress()),
          phase: "failed",
          error: txt || `HTTP ${r.status}`,
        }));
        return;
      }
      // 立即拍一次，把按钮切到 running 状态，不等下一个 tick
      await fetchProgress();
    } finally {
      setTriggering(false);
    }
  }

  const pct = progress && progress.total > 0
    ? Math.min(100, Math.round((progress.done / progress.total) * 100))
    : 0;

  return (
    <div className="flex items-center gap-3">
      {progress && (progress.phase === "done" || progress.phase === "failed") && (
        <SyncResult progress={progress} />
      )}
      {running && progress && (
        <SyncRunning progress={progress} pct={pct} />
      )}
      <button
        type="button"
        onClick={startSync}
        disabled={running || triggering}
        className={`rounded-lg px-3 py-1.5 text-sm font-medium transition-colors
          ${running || triggering
            ? "cursor-not-allowed bg-zinc-200 text-zinc-500 dark:bg-zinc-700"
            : "bg-blue-600 text-white hover:bg-blue-700"}
        `}
      >
        {running ? "同步中…" : "同步飞书通讯录"}
      </button>
    </div>
  );
}

function SyncRunning({ progress, pct }: { progress: Progress; pct: number }) {
  return (
    <div className="flex min-w-[240px] items-center gap-2 text-xs text-zinc-600 dark:text-zinc-300">
      <span className="whitespace-nowrap">{PHASE_LABEL[progress.phase]}</span>
      <div className="relative h-2 w-32 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-700">
        <div
          className="absolute inset-y-0 left-0 bg-blue-500 transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="tabular-nums">
        {progress.total > 0 ? `${progress.done}/${progress.total}` : "…"}
      </span>
    </div>
  );
}

function SyncResult({ progress }: { progress: Progress }) {
  if (progress.phase === "failed") {
    return (
      <span className="rounded-md bg-red-50 px-2 py-1 text-xs text-red-600 dark:bg-red-900/30 dark:text-red-400">
        同步失败：{progress.error || "未知错误"}
      </span>
    );
  }
  const secs = (progress.duration_ms / 1000).toFixed(1);
  const { departments: d, users: u } = progress;
  return (
    <div className="flex items-center gap-2 text-xs text-zinc-600 dark:text-zinc-300">
      <span className="rounded-md bg-emerald-50 px-2 py-1 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400">
        ✓ 用时 {secs}s
      </span>
      <Chip label="部门" created={d.created} updated={d.updated} deleted={d.deleted} />
      <Chip label="用户" created={u.created} updated={u.updated} deleted={u.deleted} />
    </div>
  );
}

function Chip({
  label,
  created,
  updated,
  deleted,
}: {
  label: string;
  created: number;
  updated: number;
  deleted: number;
}) {
  return (
    <span className="rounded-md border border-zinc-200 px-2 py-1 dark:border-zinc-700">
      {label}
      <span className="ml-1 text-emerald-600 dark:text-emerald-400">+{created}</span>
      <span className="ml-1 text-amber-600 dark:text-amber-400">~{updated}</span>
      <span className="ml-1 text-rose-600 dark:text-rose-400">-{deleted}</span>
    </span>
  );
}

function blankProgress(): Progress {
  return {
    job_id: "",
    phase: "idle",
    started_at: new Date().toISOString(),
    duration_ms: 0,
    done: 0,
    total: 0,
    departments: { created: 0, updated: 0, deleted: 0, total: 0 },
    users: { created: 0, updated: 0, deleted: 0, total: 0 },
  };
}
