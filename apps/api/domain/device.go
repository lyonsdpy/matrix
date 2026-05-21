package domain

type Device struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	MIP         string `json:"mip"` // gqlgen 通过 json tag 把 MIP 映射到 GraphQL 字段 mip
	LoginUser   string `json:"login_user"`
	LoginMethod string `json:"login_method"`
	LoginPasswd string `json:"login_passwd"`
}

type DeviceLink struct {
	Target   *Device        `json:"target"`
	Relation *GraphRelation `json:"graph_relation"`
}

// DeviceConnection 是设备列表的分页容器，对应 GraphQL DeviceConnection 类型。
type DeviceConnection struct {
	Nodes       []*Device
	HasNextPage bool
	EndCursor   *string
}
