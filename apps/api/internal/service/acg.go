package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// ACGService 终端合规检查（Endpoint Compliance）。
//
// 由后端反向请求客户端 IP 的安全客户端端口，按响应状态码 + JSON body 判定。
// 浏览器侧因 CORS/mixed-content/no-cors opaque 等限制，无法直接读取目标
// 端口的状态码和 body，只能交给后端来做。
//
// 当前探测目标：
//
//	POST https://[client_ip]:8445/both_way/communication
//
// 判定标准：HTTP 200 且响应 body 是合法 JSON、字段 errorCode == 200。
// 8445 端口通常是自签证书，TLS 验证默认关闭（仅探测可达性，不传敏感数据）。
//
// 历史背景：先前探测目标是亚信 OfficeScan 的 127.0.0.1:16721，按
// `Server: OfficeScan Client` 头判定；策略变更后改为统一探测 8445，
// Mac 与 Windows 都需做检查，仅移动端免检。
type ACGService struct {
	httpClient  *http.Client
	edrPort     int
	requestPath string
	// expectedCode 期望响应 JSON 中 errorCode 字段的值
	expectedCode int
}

const (
	// 安全客户端 HTTPS 监听端口
	defaultEDRPort = 8445
	defaultEDRPath = "/both_way/communication"
	// 客户端正确响应时返回的 errorCode，业务上是"参数错误"，但对探测来说
	// 关键信号是"端口在监听、客户端能正常解析请求并回 200"。
	expectedErrorCode = 200
)

// NewACGService 构造服务，timeout 控制对客户端探测端口的请求超时。
func NewACGService(timeout time.Duration) *ACGService {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &ACGService{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				// 关闭长连接复用，避免向不同 PC 客户端连接被串用
				DisableKeepAlives: true,
				// 8445 端口通常是自签证书，跳过验证。安全说明：探测仅校验
				// 端口可达性与响应结构，不携带任何敏感数据，TLS 信任链与否
				// 不影响业务安全模型。
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			// 不跟随重定向，按原始响应判定
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		edrPort:      defaultEDRPort,
		requestPath:  defaultEDRPath,
		expectedCode: expectedErrorCode,
	}
}

// EDRCheckResult 检查结果；Pass=true 表示终端已合规。
type EDRCheckResult struct {
	Pass   bool   `json:"pass"`
	Reason string `json:"reason,omitempty"` // 失败时回填具体原因，便于排查
}

// edrProbeResponse 与客户端返回结构对齐
// 示例：{"errorCode": 200, "errorInfo": "Error Params"}
type edrProbeResponse struct {
	ErrorCode int    `json:"errorCode"`
	ErrorInfo string `json:"errorInfo"`
}

// 响应 body 大小上限，防止异常端口回写超大流量耗尽内存
const maxProbeBodyBytes = 64 * 1024

// CheckEDR 向 clientIP:8445/both_way/communication 发空 POST，按 status 200 +
// JSON errorCode == 200 判定。
//
// 任何网络层失败（连接拒绝/超时/TLS 握手失败/不可达）都返回 Pass=false 而非
// error，error 仅用于"调用方传参非法"等编程错误，避免上层把"终端未安装"当成
// 系统故障。
func (s *ACGService) CheckEDR(ctx context.Context, clientIP string) (EDRCheckResult, error) {
	raw := strings.TrimSpace(clientIP)
	if raw == "" {
		return EDRCheckResult{}, errors.New("client ip is empty")
	}
	parsed := net.ParseIP(raw)
	if parsed == nil {
		return EDRCheckResult{}, fmt.Errorf("invalid client ip: %s", raw)
	}
	// 规范化：dual-stack 监听下浏览器走 IPv6 loopback 会让 ClientIP 拿到 ::1，
	// IPv4-mapped IPv6（::ffff:1.2.3.4）也常见。安全客户端通常仅监听 IPv4 端口，
	// 这里统一翻译回 IPv4。
	ip := normalizeProbeIP(parsed)

	url := fmt.Sprintf("https://%s:%d%s", joinHostPort(ip), s.edrPort, s.requestPath)
	// 发空 POST body：客户端响应的 errorInfo="Error Params" 正是空参数引起的，
	// 这恰好是我们用于判定端口存活的稳定信号。
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(""))
	if err != nil {
		return EDRCheckResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		// 网络层错误：连接拒绝/超时/TLS 失败/不可达，等同于"客户端未安装或未运行"
		return EDRCheckResult{Pass: false, Reason: classifyNetErr(err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return EDRCheckResult{
			Pass:   false,
			Reason: fmt.Sprintf("unexpected status: %d", resp.StatusCode),
		}, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxProbeBodyBytes))
	if err != nil {
		return EDRCheckResult{Pass: false, Reason: "read body failed: " + err.Error()}, nil
	}

	var parsedBody edrProbeResponse
	if err := json.Unmarshal(body, &parsedBody); err != nil {
		return EDRCheckResult{
			Pass:   false,
			Reason: fmt.Sprintf("invalid response body: %s", truncateForLog(body)),
		}, nil
	}
	if parsedBody.ErrorCode != s.expectedCode {
		return EDRCheckResult{
			Pass:   false,
			Reason: fmt.Sprintf("unexpected errorCode: %d (info=%q)", parsedBody.ErrorCode, parsedBody.ErrorInfo),
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

// normalizeProbeIP 把客户端 IP 翻译成适合探测安全客户端的地址。
// - IPv6 loopback (::1) → 127.0.0.1：dual-stack 监听下浏览器常走 ::1，但客户端监听 IPv4
// - IPv4-mapped IPv6 (::ffff:1.2.3.4) → 1.2.3.4：剥外壳取真实 IPv4
// - 纯 IPv6（既非 loopback 也非 mapped）：保留，由网络层决定可达性
func normalizeProbeIP(ip net.IP) string {
	if ip.IsLoopback() {
		return "127.0.0.1"
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.String()
}

// classifyNetErr 给前端的提示用，区分超时与连接拒绝便于运维定位
func classifyNetErr(err error) string {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "endpoint probe timeout"
	}
	return "endpoint probe failed: " + err.Error()
}

// truncateForLog 截断异常响应 body 便于排查，避免日志被巨型 body 撑爆
func truncateForLog(b []byte) string {
	const max = 200
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "...(truncated)"
}
