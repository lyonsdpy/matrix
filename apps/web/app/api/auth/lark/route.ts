import { NextResponse } from "next/server";
import crypto from "crypto";

const LARK_APP_ID = process.env.LARK_APP_ID!;
const LARK_REDIRECT_URI = process.env.LARK_REDIRECT_URI!;

// GET /api/auth/lark
// 生成 CSRF state，写入 httpOnly cookie，然后 302 到飞书授权页。
// 浏览器点击"飞书登录"后直接访问此地址即可启动 OAuth 流程。
export async function GET() {
  const state = crypto.randomBytes(16).toString("hex");

  const params = new URLSearchParams({
    app_id: LARK_APP_ID,
    redirect_uri: LARK_REDIRECT_URI,
    state,
  });
  const authorizeURL = `https://open.feishu.cn/open-apis/authen/v1/authorize?${params}`;

  const response = NextResponse.redirect(authorizeURL);
  // sameSite=lax：飞书回调是跨站 redirect，strict 模式下 cookie 不会随之带回
  response.cookies.set("lark_oauth_state", state, {
    httpOnly: true,
    sameSite: "lax",
    maxAge: 60 * 5, // 5 分钟，超时则 state 失效
    path: "/",
  });
  return response;
}
