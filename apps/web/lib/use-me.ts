"use client";

import { useEffect, useState } from "react";
import type { MePermissions } from "./types";

// 模块级缓存 + 单飞 in-flight Promise：
// - 整个浏览器 session 内只拉一次 /me/permissions，避免每个组件各自请求
// - 多组件并发 mount 时共享同一个 inflight，不发起重复请求
// - 授权变更后调 invalidateMe() 强制下次重拉
let cached: MePermissions | null = null;
let inflight: Promise<MePermissions> | null = null;

async function loadMe(): Promise<MePermissions> {
  if (cached) return cached;
  if (!inflight) {
    inflight = fetch("/api/proxy/me/permissions")
      .then((r) => {
        if (!r.ok) throw new Error(`me/permissions ${r.status}`);
        return r.json() as Promise<MePermissions>;
      })
      .then((d) => {
        cached = d;
        return d;
      })
      .finally(() => {
        inflight = null;
      });
  }
  return inflight;
}

export function invalidateMe() {
  cached = null;
}

// useMe 给前端提供"我有哪些权限"。
//   - permissions: Set<code>
//   - hasPerm(code): 按钮/菜单显隐用
//   - isAdmin: admin 用户后端返回全部权限码且 is_admin=true
//   - loading: 首次拉取期间为 true
//
// 注意：未授权用户的 /api/proxy/me/permissions 走 proxy 已带 401 跳 /login，
// 这里若失败默认空集合，组件会渲染"无权限"状态
export function useMe() {
  const [data, setData] = useState<MePermissions | null>(cached);

  useEffect(() => {
    if (!data) {
      loadMe()
        .then(setData)
        .catch(() =>
          setData({ user_id: "", username: "", is_admin: false, permissions: [] }),
        );
    }
  }, [data]);

  const set = new Set(data?.permissions ?? []);
  return {
    me: data,
    permissions: set,
    hasPerm: (code: string) => set.has(code),
    isAdmin: Boolean(data?.is_admin),
    loading: !data,
  };
}
