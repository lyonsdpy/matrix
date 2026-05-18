// Package aisacg 封装亚信上网行为管理（ACG）设备的 HTTP 客户端及业务实现。
package aisacg

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

// ErrMissingCredentials 在 NewClient 时 Host/Username/Password 任一为空时返回。
var ErrMissingCredentials = errors.New("aisacg: missing host, username, or password")

// ErrAPIFailure 在 API 返回 code != 1 时返回。
var ErrAPIFailure = errors.New("aisacg: api failure")

// apiResponse 是 infra 层内部使用的响应信封，Code 兼容 int 和 string 两种 JSON 格式。
// 真机观测：部分接口返回 "code":"1"（字符串），部分返回 "code":1（整数）。
type apiResponse struct {
	Code flexCode        `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// flexCode 兼容 API 返回的整数或字符串形式的响应码。
type flexCode int

func (c *flexCode) UnmarshalJSON(data []byte) error {
	s := string(data)
	// 去除引号（字符串形式 "1"）
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("aisacg: parse response code %q: %w", string(data), err)
	}
	*c = flexCode(n)
	return nil
}

// Config 亚信 ACG 设备连接参数，由调用方从配置文件读取后以入参形式传入。
type Config struct {
	Host               string // 设备 IP 或域名（不含协议前缀）
	Username           string // 管理账号
	Password           string // 管理密码
	InsecureSkipVerify bool   // 是否跳过 TLS 证书校验（设备使用自签名证书时设为 true）
}

// Client 封装对亚信 ACG 设备的 HTTP 请求，内置 Basic Auth。
type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
	logger     *zap.Logger
}

// NewClient 构造 Client。cfg.Host/Username/Password 任一为空时返回 ErrMissingCredentials。
// InsecureSkipVerify 默认建议设为 true（设备通常使用自签名证书）。
func NewClient(cfg Config, logger *zap.Logger) (*Client, error) {
	if cfg.Host == "" || cfg.Username == "" || cfg.Password == "" {
		return nil, ErrMissingCredentials
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // 设备使用自签名证书，由配置控制
		},
	}

	return &Client{
		httpClient: &http.Client{Transport: transport},
		baseURL:    "https://" + cfg.Host + "/api/v3/Objects",
		username:   cfg.Username,
		password:   cfg.Password,
		logger:     logger,
	}, nil
}

// do 发送 HTTP 请求并解析响应信封，返回 data 字段的原始 JSON。
// body 非 nil 时序列化为 JSON 请求体；为 nil 时不设置请求体。
func (c *Client) do(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("aisacg: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("aisacg: create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aisacg: send request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn("aisacg: close response body", zap.Error(closeErr))
		}
	}()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("aisacg: read response body: %w", err)
	}

	var envelope apiResponse
	if err := json.Unmarshal(respBytes, &envelope); err != nil {
		return nil, fmt.Errorf("aisacg: unmarshal response envelope: %w", err)
	}

	if envelope.Code != 1 {
		return nil, fmt.Errorf("aisacg: code=%d msg=%s: %w", envelope.Code, envelope.Msg, ErrAPIFailure)
	}

	return envelope.Data, nil
}
