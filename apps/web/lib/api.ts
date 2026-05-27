import { cookies } from "next/headers";

const GO_API_URL = process.env.GO_API_URL!;

// serverFetch 在 Next.js Server Component / Route Handler 中调用 Go API。
// 从 httpOnly cookie 取出 JWT，附加到 Authorization header 发送。
// 浏览器永远不持有明文 JWT，只持有 Next.js 域的加密 cookie。
export async function serverFetch(path: string, init?: RequestInit): Promise<Response> {
  const cookieStore = await cookies();
  const token = cookieStore.get("matrix_session")?.value;

  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...(init?.headers as Record<string, string>),
  };
  if (token) {
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }

  return fetch(`${GO_API_URL}${path}`, { ...init, headers });
}
