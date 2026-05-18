package core

type Device struct {
	ID   string // 设备ID
	Name string // 设备名称
	Type string // 设备类型
	MIP  string // 管理IP
}

type IP struct {
	Addr string
	Mask string
	Cidr string
}
