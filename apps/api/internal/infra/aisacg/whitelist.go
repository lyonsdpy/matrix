package aisacg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	domain "matrix/api/domain/aisacg"

	"go.uber.org/zap"
)

// 编译期 interface 合规检查。
var _ domain.WhitelistFetcher = (*whitelistFetcher)(nil)

type whitelistFetcher struct {
	client *Client
	logger *zap.Logger
}

// NewWhitelistFetcher 构造 WhitelistFetcher 实现。
func NewWhitelistFetcher(client *Client, logger *zap.Logger) domain.WhitelistFetcher {
	return &whitelistFetcher{client: client, logger: logger}
}

// jsonBool 兼容 ACG 设备返回 "0"/"1" 字符串形式的 bool 值。
type jsonBool bool

func (b *jsonBool) UnmarshalJSON(data []byte) error {
	// 尝试原生 bool
	var native bool
	if err := json.Unmarshal(data, &native); err == nil {
		*b = jsonBool(native)
		return nil
	}
	// 兼容字符串 "0" / "1"
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("jsonBool: unexpected token %s", data)
	}
	switch s {
	case "1", "true":
		*b = true
	case "0", "false":
		*b = false
	default:
		return fmt.Errorf("jsonBool: unexpected value %q", s)
	}
	return nil
}

// whitelistItemResp 是 GET /Policies/GlobalWhitelist 响应数组元素的内部映射类型。
type whitelistItemResp struct {
	Enable jsonBool `json:"enable"`
	Name   string `json:"name"`
	Desc   string `json:"desc"`
	Addr   []struct {
		Address string `json:"address"`
	} `json:"addr"`
}

// ListWhitelistEntries 从 ACG 设备采集全局白名单条目列表（GET /Policies/GlobalWhitelist）。
// 使用 "/../Policies/GlobalWhitelist" 路径以跨越 baseURL 的 /Objects 前缀。
func (f *whitelistFetcher) ListWhitelistEntries(ctx context.Context) ([]domain.WhitelistEntry, error) {
	data, err := f.client.do(ctx, http.MethodGet, "/../Policies/GlobalWhitelist", nil)
	if err != nil {
		return nil, fmt.Errorf("aisacg: ListWhitelistEntries: %w", err)
	}
	var items []whitelistItemResp
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("aisacg: ListWhitelistEntries unmarshal: %w", err)
	}
	entries := make([]domain.WhitelistEntry, 0, len(items))
	for _, item := range items {
		addrs := make([]string, 0, len(item.Addr))
		for _, a := range item.Addr {
			addrs = append(addrs, a.Address)
		}
		entries = append(entries, domain.WhitelistEntry{
			Enable: bool(item.Enable),
			Name:   item.Name,
			Desc:   item.Desc,
			Addrs:  addrs,
		})
	}
	return entries, nil
}
