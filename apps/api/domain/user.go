package domain

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FeishuID string `json:"feishu_id"`
}

// SyncedUser 已从飞书同步到 Neo4j 的用户完整视图，用于用户管理查询。
// 比 User 多出 sync 写入的飞书属性，仅用于读展示，不参与图关系建模。
type SyncedUser struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	FeishuID      string   `json:"feishu_id"` // open_id
	UserID        string   `json:"user_id"`   // 飞书租户内 user_id
	Email         string   `json:"email"`
	Mobile        string   `json:"mobile"`
	Status        int      `json:"status"`
	DepartmentIDs []string `json:"department_ids"`
}

type Group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UserGroupLink 用户与组的关联边，User/Group 均为完整对象，与 DeviceLink 的 Target 对齐
type UserGroupLink struct {
	User     *User
	Group    *Group
	Relation *GraphRelation
}

// GroupGroupLink 组与组的关联(如父子关系)，Target是关联的另一个组
type GroupGroupLink struct {
	Target   *Group
	Relation *GraphRelation
}

// GroupConnection 组列表的分页容器，对应GraphQL GroupConnection
type GroupConnection struct {
	Nodes       []*Group
	HasNextPage bool
	EndCursor   *string
}
