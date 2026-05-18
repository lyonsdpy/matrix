package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPConfig HTTP 轮询采集器配置
type HTTPConfig struct {
	URL            string            `json:"url"`             // 必填
	Method         string            `json:"method"`          // 默认 GET
	Headers        map[string]string `json:"headers"`
	Body           string            `json:"body"`
	TimeoutSecs    int               `json:"timeout_secs"`    // 默认 10
	ExpectedStatus int               `json:"expected_status"` // 0 表示不校验
}

type httpCollector struct{}

func init() {
	Register(&httpCollector{})
}

func (c *httpCollector) Type() string { return "http" }
func (c *httpCollector) Description() string {
	return "定期请求 HTTP 接口并采集响应内容，适用于网络设备 API、阿里云 OpenAPI、第三方服务等"
}

func (c *httpCollector) Collect(ctx context.Context, rawConfig json.RawMessage) (*Result, error) {
	var cfg HTTPConfig
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return nil, fmt.Errorf("解析 HTTP 配置失败: %w", err)
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("url 不能为空")
	}
	if cfg.Method == "" {
		cfg.Method = "GET"
	}
	timeout := 10 * time.Second
	if cfg.TimeoutSecs > 0 {
		timeout = time.Duration(cfg.TimeoutSecs) * time.Second
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var bodyReader io.Reader
	if cfg.Body != "" {
		bodyReader = strings.NewReader(cfg.Body)
	}
	req, err := http.NewRequestWithContext(reqCtx, cfg.Method, cfg.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if cfg.ExpectedStatus != 0 && resp.StatusCode != cfg.ExpectedStatus {
		return nil, fmt.Errorf("响应状态码 %d，期望 %d，body: %.200s", resp.StatusCode, cfg.ExpectedStatus, respBody)
	}

	// 尝试保持 JSON 格式，否则退化为字符串
	var data interface{}
	if json.Valid(respBody) {
		data = json.RawMessage(respBody)
	} else {
		data = string(respBody)
	}

	return &Result{
		Data:    data,
		Summary: fmt.Sprintf("HTTP %s %s → %d, body_len=%d", cfg.Method, cfg.URL, resp.StatusCode, len(respBody)),
	}, nil
}
