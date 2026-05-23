package domain

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FeishuID string `json:"feishu_id"`
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
