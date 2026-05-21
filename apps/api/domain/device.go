package domain

type Device struct {
	ID           string `json:"id"`           // 设备ID
	Name         string `json:"name"`         // 设备名称
	Type         string `json:"type"`         // 设备类型
	TypeCN       string `json:"type_cn"`      // 设备类型(中文)
	Vendor       string `json:"vendor"`       // 设备厂商
	VendorCN     string `json:"vendor_cn"`    // 设备厂商(中文)
	MIP          string `json:"mip"`          // 管理IP
	LoginMethod  string `json:"login_method"` // 登录方式
	LoginUser    string `json:"login_user"`   // 登录用户名
	LoginPasswd  string `json:"login_passwd"` // 登录密码
	TemporalMeta        // 当前态时态元数据（版本、时间戳、软删除）
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
