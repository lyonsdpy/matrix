// Package lark 提供飞书 SDK Client 的初始化封装。
package lark

import (
	"errors"

	larkSDK "github.com/larksuite/oapi-sdk-go/v3"
)

// Client 是飞书 SDK Client 的类型别名，供调用方使用 lark.Client 而无需感知 SDK 路径。
type Client = larkSDK.Client

// Config 飞书应用凭据，由调用方（main/wire）从配置文件读取后以入参形式传入。
type Config struct {
	AppID     string // 飞书应用 App ID
	AppSecret string // 飞书应用 App Secret
}

// ErrMissingCredentials 当 AppID 或 AppSecret 为空时返回。
var ErrMissingCredentials = errors.New("lark: AppID and AppSecret must not be empty")

// NewClient 使用 AppID + AppSecret 创建飞书 SDK Client。
func NewClient(cfg Config) (*Client, error) {
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, ErrMissingCredentials
	}
	return larkSDK.NewClient(cfg.AppID, cfg.AppSecret), nil
}
