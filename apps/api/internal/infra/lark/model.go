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
type Device struct {
	DeviceID       string // 设备 ID
	DeviceName     string // 设备名称
	Platform       string // 平台（Windows/Mac/Android/iOS 等）
	UserID         string // 设备归属用户
	Status         string // 设备状态
	TrustLevel     string // 可信级别
	SerialNumber   string // 生产序列号
	OSVersion      string // 操作系统版本
	LastOnlineTime int64  // 最后在线时间（Unix 时间戳）
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
