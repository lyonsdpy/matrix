package domain

// DepartmentRef 部门的最小引用，用于路径、面包屑等列表场景。
type DepartmentRef struct {
	DepartmentID string `json:"department_id"`
	Name         string `json:"name"`
}

// DepartmentNode 部门树节点：树展示与子部门列表共用。
// HasChildren 用于树视图判断是否还能展开（避免再请一次接口才知道是叶子）。
type DepartmentNode struct {
	DepartmentID string `json:"department_id"`
	Name         string `json:"name"`
	ParentID     string `json:"parent_id"`
	MemberCount  int    `json:"member_count"`
	HasChildren  bool   `json:"has_children"`
}

// DepartmentDetail 部门详情面板用：包含路径、子部门、直属成员、递归人数。
// RecursiveMemberCount 通过 [:PARENT_OF*]+[:MEMBER_OF] 聚合，体现图特性。
type DepartmentDetail struct {
	DepartmentID         string            `json:"department_id"`
	Name                 string            `json:"name"`
	ParentID             string            `json:"parent_id"`
	LeaderUserID         string            `json:"leader_user_id"`
	MemberCount          int               `json:"member_count"`
	RecursiveMemberCount int               `json:"recursive_member_count"`
	Path                 []*DepartmentRef  `json:"path"`
	Children             []*DepartmentNode `json:"children"`
	DirectMembers        []*SyncedUser     `json:"direct_members"`
}
