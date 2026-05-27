// 已同步的飞书用户，对应后端 domain.SyncedUser。
export interface SyncedUser {
  id: string;
  name: string;
  feishu_id: string; // open_id
  user_id: string;
  email: string;
  mobile: string;
  status: number;
  department_ids: string[];
}

// 用户列表查询响应，对应后端 service.SyncedUserList。
export interface SyncedUserList {
  users: SyncedUser[];
  has_next: boolean;
  end_cursor: string;
}
