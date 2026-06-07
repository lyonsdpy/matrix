import { ContactsClient } from "@/components/contacts/ContactsClient";
import { SyncContactsButton } from "@/components/contacts/SyncContactsButton";

// 用户管理：两栏(部门树 + 成员区) + 抽屉详情。数据源仍是飞书通讯录，故路由保留 /contacts。
// 主交互完全在 client component(ContactsClient)，本页只负责标题与挂载。
export default function ContactsPage() {
  // h-full + flex-col：占满 layout 留给 main 的高度，让 ContactsClient 用 flex-1 接管剩余空间。
  return (
    <div className="flex h-full flex-col gap-4">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold">用户管理</h1>
        <SyncContactsButton />
      </div>
      <ContactsClient />
    </div>
  );
}
