package domain

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FeishuID string `json:"feishu_id"`
}

// SyncedUser 已从飞书同步到 Neo4j 的用户完整视图，用于用户管理查询。
// 比 User 多出 sync 写入的飞书属性，仅用于读展示，不参与图关系建模。
type SyncedUser struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	FeishuID        string   `json:"feishu_id"` // open_id
	UserID          string   `json:"user_id"`   // 飞书租户内 user_id
	Email           string   `json:"email"`
	Mobile          string   `json:"mobile"`
	Status          int      `json:"status"`
	DepartmentIDs   []string `json:"department_ids"`
	DepartmentNames []string `json:"department_names"` // 通过 MEMBER_OF 边取到的部门中文名
}

// UserDepartment 用户的某个部门归属，带从顶级到该部门的完整路径，供详情面包屑展示。
type UserDepartment struct {
	DepartmentID string           `json:"department_id"`
	Name         string           `json:"name"`
	Path         []*DepartmentRef `json:"path"`
}

// UserDetail 用户详情面板：基本字段 + 所属部门(每个带完整路径) + 上级领导 + 管理部门。
// 上级领导：所属部门沿 PARENT_OF 链向上，第一个 leader_user_id 非自己的部门的 leader。
// 管理部门：leader_user_id == 本人 user_id 的部门集合。
type UserDetail struct {
	*SyncedUser
	Departments        []*UserDepartment `json:"departments"`
	Leaders            []*SyncedUser     `json:"leaders"`
	ManagedDepartments []*DepartmentRef  `json:"managed_departments"`
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
