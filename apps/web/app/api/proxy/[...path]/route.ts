import { serverFetch } from "@/lib/api";

// 通用代理：浏览器侧 Client Component 调 /api/proxy/<path>，转发到 Go API /api/v1/<path>。
// JWT 通过 serverFetch 从 httpOnly cookie 注入到 Authorization header，token 永不出 server。
//
// 透传 GET 与 POST：写操作（如触发同步）需要 POST，body 原样转发。
export async function GET(
  req: Request,
  { params }: { params: Promise<{ path: string[] }> },
) {
  const { path } = await params;
  const search = new URL(req.url).search;
  const target = `/api/v1/${path.join("/")}${search}`;
  const res = await serverFetch(target);
  return passthrough(res);
}

export async function POST(
  req: Request,
  { params }: { params: Promise<{ path: string[] }> },
) {
  const { path } = await params;
  const search = new URL(req.url).search;
  const target = `/api/v1/${path.join("/")}${search}`;
  // 触发同步等 endpoint 无 body 也兼容
  const body = await req.text();
  const res = await serverFetch(target, {
    method: "POST",
    body: body || undefined,
  });
  return passthrough(res);
}

async function passthrough(res: Response): Promise<Response> {
  const body = await res.text();
  return new Response(body, {
    status: res.status,
    headers: {
      "Content-Type": res.headers.get("Content-Type") ?? "application/json",
    },
  });
}
