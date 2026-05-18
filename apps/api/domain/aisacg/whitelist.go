package aisacg

import "context"

// WhitelistEntry ACG 全局白名单条目。
// 只读采集语义：此类型仅用于从 ACG 设备采集数据，不支持本地修改。
type WhitelistEntry struct {
	Enable bool     // 是否启用
	Name   string   // 条目名称
	Desc   string   // 描述
	Addrs  []string // 地址列表（IP 或域名）
}

// WhitelistFetcher 只读采集 ACG 全局白名单。
type WhitelistFetcher interface {
	// ListWhitelistEntries 从 ACG 设备采集全局白名单条目列表。
	ListWhitelistEntries(ctx context.Context) ([]WhitelistEntry, error)
}
