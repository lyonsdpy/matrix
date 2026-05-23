// gqlclient 是 examples 专用的轻量 GraphQL HTTP 客户端。
// 使用 cookie jar 自动维护登录态（matrix_session）。
package gqlclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
)

// Client 持有 HTTP 连接和 session，复用同一 cookie jar。
type Client struct {
	baseURL string
	http    *http.Client
}

// New 创建客户端，baseURL 形如 "http://localhost:8080"。
func New(baseURL string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Jar: jar},
	}, nil
}

// Login 调用 POST /api/v1/auth/login，成功后 matrix_session cookie 自动保存在 jar 里。
func (c *Client) Login(username, password string) error {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := c.http.Post(c.baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed %s: %s", resp.Status, b)
	}
	return nil
}

// Response 是标准 GraphQL 响应信封。
type Response struct {
	Data   json.RawMessage `json:"data"`
	Errors json.RawMessage `json:"errors,omitempty"`
}

// Print 以缩进 JSON 打印完整响应（含 data / errors 外层结构）。
func (r *Response) Print() {
	b, _ := json.MarshalIndent(r, "", "  ")
	fmt.Println(string(b))
}

// Do 发送 GraphQL 请求并返回解析后的响应。
// query 传 mutation 或 query 字符串，variables 可为 nil。
func (c *Client) Do(query string, variables map[string]any) (*Response, error) {
	payload, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/graphql", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result Response
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w\nraw: %s", err, b)
	}
	return &result, nil
}
