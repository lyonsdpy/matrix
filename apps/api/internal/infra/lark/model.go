// Package lark 定义飞书领域的纯 Go 模型与 interface，零外部依赖。
package lark

// User 飞书用户模型（组织架构同步用）。
type User struct {
	UserID        string   // 租户内唯一用户 ID（user_id）
	OpenID        string   // 应用内唯一标识（open_id），与 OAuth 扫码登录的关联键对齐
	Name          string   // 用户姓名
	Email         string   // 邮箱
	Mobile        string   // 手机号
	Status        int      // 在职状态（1=在职, 2=已冻结, 3=离职, 4=待入职, 0=未知）
	DepartmentIDs []string // 所属部门 ID 列表
}

// Department 飞书部门模型。
type Department struct {
	DepartmentID string // 部门 ID
	Name         string // 部门名称
	ParentID     string // 上级部门 ID（根部门为 "0"）
	LeaderUserID string // 部门负责人 user_id
	MemberCount  int    // 部门直属成员数
	Status       int    // 部门状态
}

// Device 飞书安全合规设备模型。
// CurrentUserID/LatestUserID 都是飞书 user_id；二者可能不同，业务上
// 当前登录(Current)和最近一次登录(Latest)用两种关系边分别表达。
type Device struct {
	DeviceID         string // 设备 ID（device_record_id）
	DeviceName       string // 设备名称
	Platform         string // device_terminal_type 物理形态编号（0=未知/1=移动端/2=桌面端）
	CurrentUserID    string // 当前登录用户 ID（飞书 user_id）
	LatestUserID     string // 最近登录用户 ID（飞书 user_id）
	Ownership        string // device_ownership 原始编号（1/2 公司/个人）
	TrustLevel       string // device_status 原始编号（可信状态）
	Certification    string // certification_level 原始编号（认证方式）
	SerialNumber     string // 生产序列号
	DiskSerialNumber string // 硬盘序列号
	BoardUUID        string // 主板 UUID
	MACAddress       string // MAC 地址
	Model            string // 设备型号
	OSCode           string // device_system 原始编号（操作系统）
	Version          string // 版本号
	IsManaged        bool   // 是否为受管控设备
	MDMDeviceID      string // MDM 设备 ID
	MDMProvider      string // MDM 厂商名称
	LastOnlineTime   int64  // 最后在线时间（Unix 时间戳）

	// Status / OSVersion 保留旧字段以兼容历史调用方；新代码请使用 Ownership / OSCode。
	Status    string
	OSVersion string
}

// Message 飞书消息模型（收发通用）。
type Message struct {
	MessageID  string // 消息 ID
	ChatID     string // 会话 ID
	ChatType   string // 会话类型（"p2p" | "group"）
	SenderID   string // 发送者 user_id
	MsgType    string // 消息类型（"text", "interactive" 等）
	Content    string // 消息内容（JSON 字符串）
	CreateTime string // 创建时间
}

// CardAction 卡片回传交互动作。
type CardAction struct {
	UserID    string            // 操作用户 user_id
	OpenID    string            // 操作用户 open_id
	Action    string            // 动作标识（按钮 action value）
	FormValue map[string]string // 表单值（如有）
	ChatID    string            // 所在会话 ID
}
