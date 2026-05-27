import { NextRequest, NextResponse } from "next/server";

// 路由守卫（Next 16 proxy 约定，原 middleware）：登录态由 matrix_session（httpOnly cookie）决定。
// - 未登录访问受保护页面 → 跳 /login
// - 已登录访问 /login → 跳 /users（避免重复登录）
// matcher 已排除 /api/auth、_next、静态资源，OAuth 流程不受影响。
export function proxy(req: NextRequest) {
  const hasSession = Boolean(req.cookies.get("matrix_session")?.value);
  const isLoginPage = req.nextUrl.pathname === "/login";

  if (!hasSession && !isLoginPage) {
    return NextResponse.redirect(new URL("/login", req.url));
  }
  if (hasSession && isLoginPage) {
    return NextResponse.redirect(new URL("/users", req.url));
  }
  return NextResponse.next();
}

export const config = {
  // 拦截除以下之外的所有路由：API 认证回调、Next 内部资源、静态文件
  matcher: ["/((?!api/auth|_next/static|_next/image|favicon.ico).*)"],
};
