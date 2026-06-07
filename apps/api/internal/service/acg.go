package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// ACGService 终端合规检查（Endpoint Compliance）。
//
// 原本由 acg-checker 浏览器端直接探测本机 16721 端口，但浏览器在
// no-cors 模式下读不到 Status 和 Server 头，只能凭 fetch 是否 resolve
// 粗糙判定，存在误报。改为由后端反向请求客户端 IP 的 OfficeScan 端口，
// 拿到完整 Response 后严格按 503 + Server: OfficeScan Client 判定。
type ACGService struct {
	httpClient    *http.Client
	edrPort       int
	requestPath   string
	expectedSrv   string
	expectedCode  int
}

const (
	// OfficeScan Client 监听端口，固定值；如需可改为配置项再迁移
	defaultEDRPort   = 16721
	defaultEDRPath   = "/"
	expectedServer   = "OfficeScan Client"
	expectedHTTPCode = http.StatusServiceUnavailable // 503
)

// NewACGService 构造服务，timeout 控制对客户端 16721 的请求超时。
func NewACGService(timeout time.Duration) *ACGService {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &ACGService{
		// 自定义 Transport 关闭长连接复用，避免向不同 PC 客户端连接被串用
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
			// 不跟随重定向，按原始响应判定
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		edrPort:      defaultEDRPort,
		requestPath:  defaultEDRPath,
		expectedSrv:  expectedServer,
		expectedCode: expectedHTTPCode,
	}
}

// EDRCheckResult 检查结果；Pass=true 表示终端已合规。
type EDRCheckResult struct {
	Pass   bool   `json:"pass"`
	Reason string `json:"reason,omitempty"` // 失败时回填具体原因，便于排查
}

// CheckEDR 向 clientIP:16721 发 GET，按 503 + Server: OfficeScan Client 判定。
//
// 任何网络层失败（连接拒绝/超时/解析失败）都返回 Pass=false 而非 error，
// error 仅用于"调用方传参非法"等编程错误，避免上层把"终端未安装"当成系统故障。
func (s *ACGService) CheckEDR(ctx context.Context, clientIP string) (EDRCheckResult, error) {
	ip := strings.TrimSpace(clientIP)
	if ip == "" {
		return EDRCheckResult{}, errors.New("client ip is empty")
	}
	if net.ParseIP(ip) == nil {
		return EDRCheckResult{}, fmt.Errorf("invalid client ip: %s", ip)
	}

	url := fmt.Sprintf("http://%s:%d%s", joinHostPort(ip), s.edrPort, s.requestPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return EDRCheckResult{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		// 网络层错误：连接拒绝/超时/不可达，等同于"客户端未安装或未运行 EDR"
		return EDRCheckResult{Pass: false, Reason: classifyNetErr(err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != s.expectedCode {
		return EDRCheckResult{
			Pass:   false,
			Reason: fmt.Sprintf("unexpected status %d", resp.StatusCode),
		}, nil
	}
	server := resp.Header.Get("Server")
	// 大小写不敏感匹配，OfficeScan 不同版本可能略有差异
	if !strings.Contains(strings.ToLower(server), strings.ToLower(s.expectedSrv)) {
		return EDRCheckResult{
			Pass:   false,
			Reason: fmt.Sprintf("unexpected server header: %q", server),
		}, nil
	}
	return EDRCheckResult{Pass: true}, nil
}

// joinHostPort 对 IPv6 形如 ::1 自动加方括号；IPv4 原样返回
func joinHostPort(ip string) string {
	if strings.Contains(ip, ":") {
		return "[" + ip + "]"
	}
	return ip
}

// classifyNetErr 给前端的提示用，区分超时与连接拒绝便于运维定位
func classifyNetErr(err error) string {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "endpoint probe timeout"
	}
	return "endpoint probe failed: " + err.Error()
}
