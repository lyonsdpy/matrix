import { NextResponse } from "next/server";

// POST /api/auth/logout
// 清除 session cookie，完成登出。Go API 是无状态 JWT，无需通知服务端。
export async function POST() {
  const response = NextResponse.json({ ok: true });
  response.cookies.delete("matrix_session");
  return response;
}