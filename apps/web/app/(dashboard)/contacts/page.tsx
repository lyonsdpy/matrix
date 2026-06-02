import { ContactsClient } from "@/components/contacts/ContactsClient";
import { SyncContactsButton } from "@/components/contacts/SyncContactsButton";

// 通讯录管理：两栏(部门树 + 成员区) + 抽屉详情。
// 主交互完全在 client component(ContactsClient)，本页只负责标题与挂载。
export default function ContactsPage() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold">通讯录管理</h1>
        <SyncContactsButton />
      </div>
      <ContactsClient />
    </div>
  );
}
