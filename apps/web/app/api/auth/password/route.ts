import { NextRequest, NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL!;
const JWT_EXPIRY_HOURS = parseInt(process.env.JWT_EXPIRY_HOURS ?? "24", 10);

// POST /api/auth/password
// 账号密码登录：兜底通道，主要供 admin 本地账号在飞书 OAuth 不可用时登录。
// 流程：透传 username/password 到 Go API → 拿到 JWT → 写 matrix_session cookie。
// cookie 写法与飞书 callback 保持一致（同 sameSite/httpOnly/maxAge），确保 proxy.ts 守卫无差异。
export async function POST(request: NextRequest) {
  let body: { username?: string; password?: string };
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: "invalid request" }, { status: 400 });
  }
  if (!body.username || !body.password) {
    return NextResponse.json({ error: "缺少账号或密码" }, { status: 400 });
  }

  let token: string;
  try {
    const res = await fetch(`${GO_API_URL}/api/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username: body.username, password: body.password }),
    });
    if (res.status === 401) {
      return NextResponse.json({ error: "账号或密码错误" }, { status: 401 });
    }
    if (!res.ok) {
      return NextResponse.json({ error: "登录失败" }, { status: res.status });
    }
    const data = await res.json();
    token = data.token;
  } catch {
    return NextResponse.json({ error: "服务暂不可用" }, { status: 502 });
  }

  // 写入 session cookie，httpOnly 确保 JS 不可读，写法对齐飞书 callback
  const response = NextResponse.json({ ok: true });
  response.cookies.set("matrix_session", token, {
    httpOnly: true,
    sameSite: "lax",
    maxAge: JWT_EXPIRY_HOURS * 60 * 60,
    path: "/",
    secure: process.env.HTTPS_ONLY === "true",
  });
  return response;
}
