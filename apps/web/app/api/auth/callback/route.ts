import { NextRequest, NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL!;
const INTERNAL_SECRET = process.env.INTERNAL_SECRET!;
const JWT_EXPIRY_HOURS = parseInt(process.env.JWT_EXPIRY_HOURS ?? "24", 10);

// GET /api/auth/callback?code=&state=
// 飞书 OAuth 回调地址，由飞书开放平台配置指向此处。
// 流程：验证 state → 调 Go API exchange → 写 session cookie → 302 首页。
export async function GET(request: NextRequest) {
  // 跳转统一用回调地址的 origin，而非 request.url 的 host：
  // 反向代理链路里 Host 可能被改写成内网地址（如 localhost:8088），导致跳到打不开的地址。
  // 必须在请求时取值：LARK_REDIRECT_URI 是运行时环境变量，构建时不存在。
  const APP_ORIGIN = new URL(process.env.LARK_REDIRECT_URI!).origin;

  const { searchParams } = new URL(request.url);
  const code = searchParams.get("code");
  const state = searchParams.get("state");

  const savedState = request.cookies.get("lark_oauth_state")?.value;

  // state 不匹配则拒绝，防止 CSRF
  if (!code || !state || !savedState || state !== savedState) {
    return NextResponse.redirect(new URL("/login?error=invalid_state", APP_ORIGIN));
  }

  let token: string;
  let expiresAt: string;
  let user: { id: string; username: string; roles: string[] };

  try {
    const res = await fetch(`${GO_API_URL}/internal/auth/lark/exchange`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Internal-Secret": INTERNAL_SECRET,
      },
      body: JSON.stringify({ code }),
    });
    if (!res.ok) {
      throw new Error(`go api: ${res.status}`);
    }
    const data = await res.json();
    token = data.token;
    expiresAt = data.expires_at;
    user = data.user;
  } catch {
    return NextResponse.redirect(new URL("/login?error=auth_failed", APP_ORIGIN));
  }

  const response = NextResponse.redirect(new URL("/", APP_ORIGIN));

  // 清除临时 state cookie
  response.cookies.delete("lark_oauth_state");

  // 写入 session cookie，httpOnly 确保 JS 不可读
  // sameSite=lax：飞书回调是跨站 redirect 回流，strict 会导致跳转目标页拿不到 cookie。
  // lax 下顶级 GET 导航携带 cookie，跨站 POST/XHR 不带，CSRF 防护依然足够。
  response.cookies.set("matrix_session", token, {
    httpOnly: true,
    sameSite: "lax",
    maxAge: JWT_EXPIRY_HOURS * 60 * 60,
    path: "/",
    // 生产环境通过 HTTPS_ONLY=true 环境变量开启 secure
    secure: process.env.HTTPS_ONLY === "true",
  });

  void user; // user 信息由客户端在首页通过 /api/v1/me 等接口获取
  void expiresAt;

  return response;
}
