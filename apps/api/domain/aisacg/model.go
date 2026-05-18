// Package aisacg 定义亚信上网行为管理（ACG）的领域模型和 interface。
package aisacg

import (
	"encoding/json"
	"fmt"
	"time"
)

// ResponseEnvelope 是亚信 ACG API 的通用响应信封。
// Code == 1 表示成功。
type ResponseEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// BindItem 是 IP/MAC 绑定项。
type BindItem struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

// BindItems 是 BindItem 切片，实现 json.Unmarshaler 以处理 API 返回
// 空字符串 "" 时的特殊情况（无绑定时返回 "" 而非 []）。
type BindItems []BindItem

// UnmarshalJSON 实现 json.Unmarshaler。
// "" 和 null 均解析为 nil，JSON 数组正常解析。
func (b *BindItems) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == `""` || s == "null" {
		*b = nil
		return nil
	}
	var items []BindItem
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("aisacg: unmarshal BindItems: %w", err)
	}
	*b = items
	return nil
}

// AttributeInfo 是用户扩展属性。
type AttributeInfo struct {
	KeyName  string `json:"key_name"`
	KeyValue string `json:"key_value"`
	AttrType string `json:"attr_type"`
}

// User 是亚信 ACG 用户查询结果模型，对应 UserSearch API 返回结构。
type User struct {
	Path                string          `json:"path"`
	Name                string          `json:"name"`
	Enable              string          `json:"enable"`
	Description         string          `json:"description"`
	GroupType           string          `json:"group_type"`
	PhoneNum            string          `json:"phone_num"`
	EmailAddr           string          `json:"email_addr"`
	EnableLocalPassword string          `json:"enable_local_password"`
	ExpireSetting       string          `json:"expire_setting"`
	ExpireDate          string          `json:"expire_date"`
	Type                string          `json:"type"`
	Source              string          `json:"source"`
	Parent              string          `json:"parent"`
	BindInclude         BindItems       `json:"bind_include"`
	BindExclude         BindItems       `json:"bind_exclude"`
	AttributeInfo       []AttributeInfo `json:"attribute_info"`
}

// UserUpdate 是用户创建/更新请求模型。
// 注意：PasswordConfirm 的 JSON key 为 password_comfirm（API 原始拼写错误）。
type UserUpdate struct {
	Name                string          `json:"name"`
	Enable              string          `json:"enable"`
	EnableLocalPassword string          `json:"enable_local_password"`
	ExpireSetting       string          `json:"expire_setting"`
	ParentGroupPath     string          `json:"parent_group_path"`
	Alias               string          `json:"alias,omitempty"`
	Description         string          `json:"description,omitempty"`
	Password            string          `json:"password,omitempty"`
	PasswordConfirm     string          `json:"password_comfirm,omitempty"`
	PhoneNum            string          `json:"phone_num,omitempty"`
	EmailAddr           string          `json:"email_addr,omitempty"`
	AttributeInfo       []AttributeInfo `json:"attribute_info,omitempty"`
	BindInclude         BindItems       `json:"bind_include,omitempty"`
	BindExclude         BindItems       `json:"bind_exclude,omitempty"`
}

// UserOnline 是亚信 ACG 在线用户模型。
type UserOnline struct {
	Name                  string `json:"name"`
	Description           string `json:"description"`
	GroupName             string `json:"group_name"`
	IP                    string `json:"ip"`
	MAC                   string `json:"mac"`
	LoginTime             string `json:"login_time"`
	OnlineTime            string `json:"online_time"`
	FreezeEnable          string `json:"freeze_enable"`
	ExpireTime            string `json:"expire_time"`
	Platform              string `json:"platform"`
	System                string `json:"system"`
	Device                string `json:"device"`
	Supplier              string `json:"supplier"`
	ComplianceCheckResult string `json:"compliance_check_result"`
}

// OnlineTotal 是在线用户统计。
type OnlineTotal struct {
	TotalOnline string `json:"total_online"`
	AnonyOnline string `json:"anony_online"`
}

// OnlineTreeNode 是在线用户组织树节点。
type OnlineTreeNode struct {
	Path          string `json:"path"`
	Name          string `json:"name"`
	OnlineNumbers string `json:"online_numbers"`
}

// ComplianceStatus 合规状态。
type ComplianceStatus string

const (
	ComplianceUnchecked    ComplianceStatus = "unchecked"     // 未检查
	ComplianceCompliant    ComplianceStatus = "compliant"     // 合规
	ComplianceNonCompliant ComplianceStatus = "non_compliant" // 不合规（ACG 上网违规）
	ComplianceBlacklistHit ComplianceStatus = "blacklist_hit" // 黑名单命中（软件黑名单违规）
)

// ViolationRecord 单条违规记录。
type ViolationRecord struct {
	ID           string           // UUID 主键
	EmployeeID   string           // 关联 Employee UUID
	EmployeeName string           // 冗余姓名
	Office       string           // 所属办公室
	IP           string           // 违规时的 IP
	MAC          string           // 违规时的 MAC
	Status       ComplianceStatus // 合规状态
	DetectedAt   time.Time        // 检测时间
}

// OfficeComplianceReport 单个办公室的合规统计报告。
//
// 不变式：TotalOnline == CompliantCount + ViolationCount + BlacklistHitCount + UncheckedCount
type OfficeComplianceReport struct {
	Office            string            // 办公室名称
	TotalOnline       int               // 在线总人数
	CompliantCount    int               // 合规人数
	ViolationCount    int               // ACG 上网违规人数
	BlacklistHitCount int               // 软件黑名单命中人数
	UncheckedCount    int               // 未检查人数
	Violations        []ViolationRecord // 违规明细
	GeneratedAt       time.Time         // 报告生成时间
}
