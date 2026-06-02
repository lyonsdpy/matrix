import { redirect } from "next/navigation";

// /users 已并入"通讯录管理"，重定向到新页面。保留以兼容旧书签。
export default function UsersRedirect() {
  redirect("/contacts");
}
