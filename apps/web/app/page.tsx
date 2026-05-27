import { redirect } from "next/navigation";

// 根路由直接进入用户管理（未登录会被 middleware 拦到 /login）。
export default function Home() {
  redirect("/users");
}
