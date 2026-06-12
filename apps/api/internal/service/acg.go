package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"matrix/api/pkg/log"
)

// ACGService 终端合规检查（Endpoint Compliance）。
//
// 由后端反向连接客户端 IP 的安全客户端端口（8445），通过"能否完成 TLS 握手 +
// 服务端证书是否属于已知安全客户端厂商"判定终端是否安装并运行了 EDR。
//
// 当前判定：
//
//	对 https://[client_ip]:8445 发起 TLS 握手；握手成功且服务端证书
//	Subject.Organization ∈ {ais, asiainfo-sec} 即判定合规。
//
// 8445 是自签证书，TLS 验证默认关闭（仅做可达性 + 证书指纹识别，不传敏感数据）。
//
// 历史背景：
//   - 最早探测亚信 OfficeScan 的 127.0.0.1:16721，按 `Server: OfficeScan Client` 头判定；
//   - 之后改为对 8445 的 /both_way/communication 发空 POST，按 HTTP 200 + JSON
//     errorCode==200 判定。
//   - 现状：厂商新版客户端（证书 CN=www.asiainfo-sec.com，仅收 TLS1.3）改了
//     /both_way/communication 的行为——对空 `{}` 请求不再返回 errorCode:200，而是
//     直接挂起不回（实测其 `GET /` 正常回 404，仅该路由 hang），继续依赖该 HTTP
//     响应会让新版终端探测超时、被误判为"未安装"。因此判活信号下沉到 TLS 层：
//     旧版（CN=SecureHTTPServer/O=ais）与新版（CN=www.asiainfo-sec.com/O=asiainfo-sec）
//     都能完成 TLS 握手并给出可识别证书，按证书 Organization 判定即可统一覆盖两套实现。
type ACGService struct {
	dialer  *tls.Dialer
	edrPort int
	// expectedOrgs 已知安全客户端证书的 Subject.Organization 白名单
	expectedOrgs []string
}

// 安全客户端 HTTPS 监听端口
const defaultEDRPort = 8445

// knownEDROrgs 已知安全客户端自签证书的 Organization 取值：旧版客户端为 "ais"
// （CN=SecureHTTPServer），新版为 "asiainfo-sec"（CN=www.asiainfo-sec.com）。
// 后续厂商若再换证书，需同步在此补充。
var knownEDROrgs = []string{"ais", "asiainfo-sec"}

// NewACGService 构造服务，timeout 控制对客户端 8445 端口的 TLS 握手超时。
func NewACGService(timeout time.Duration) *ACGService {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &ACGService{
		dialer: &tls.Dialer{
			NetDialer: &net.Dialer{Timeout: timeout},
			// 8445 是自签证书，跳过验证。这里只取证书 Organization 做厂商识别，
			// 不依赖 TLS 信任链，探测也不携带敏感数据。
			Config: &tls.Config{InsecureSkipVerify: true},
		},
		edrPort:      defaultEDRPort,
		expectedOrgs: knownEDROrgs,
	}
}

// EDRCheckResult 检查结果；Pass=true 表示终端已合规。
type EDRCheckResult struct {
	Pass   bool   `json:"pass"`
	Reason string `json:"reason,omitempty"` // 失败时回填具体原因，便于排查
}

// CheckEDR 对 clientIP:8445 发起 TLS 握手，按"握手成功 + 服务端证书 Organization
// 属于已知安全客户端厂商"判定终端是否装有 EDR。
//
// 任何网络层失败（连接拒绝/超时/TLS 握手失败/不可达）都返回 Pass=false 而非
// error，error 仅用于"调用方传参非法"等编程错误，避免上层把"终端未安装"当成
// 系统故障。
//
// 关键路径全程打日志：判活方式从 HTTP 改为证书识别后，线上排查最需要区分
// "端口没开/超时"与"端口开着但证书不是这家 EDR"，二者结论完全相反，所以连接
// 失败、证书不匹配、判定通过三种结果都各打一条，并带上实际 CN/Organization 与耗时。
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
	addr := net.JoinHostPort(ip, strconv.Itoa(s.edrPort))
	startedAt := time.Now()

	// 只做 TLS 握手，不发任何应用层请求：新版客户端的 /both_way/communication
	// 会对空请求挂死，握手却始终秒级完成，是更稳的判活信号。
	conn, err := s.dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		// 网络层/握手错误：连接拒绝/超时/TLS 失败/不可达，等同于"客户端未安装或未运行"
		reason := classifyNetErr(err)
		log.Logger.Warnf("acg edr-check probe failed: addr=%s elapsed=%s reason=%s err=%v",
			addr, time.Since(startedAt), reason, err)
		return EDRCheckResult{Pass: false, Reason: reason}, nil
	}
	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		// tls.Dialer 正常返回 *tls.Conn，这里是防御性兜底
		log.Logger.Errorf("acg edr-check unexpected non-tls conn: addr=%s", addr)
		return EDRCheckResult{Pass: false, Reason: "unexpected non-tls connection"}, nil
	}
	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		log.Logger.Warnf("acg edr-check no server cert: addr=%s elapsed=%s", addr, time.Since(startedAt))
		return EDRCheckResult{Pass: false, Reason: "no server certificate"}, nil
	}

	leaf := certs[0]
	if !matchKnownOrg(leaf.Subject.Organization, s.expectedOrgs) {
		// 端口开着、TLS 也通，但证书不属于已知 EDR 厂商：可能是别的服务占了 8445，
		// 把实际 CN/Org 打出来便于判断到底连到了什么。
		log.Logger.Warnf("acg edr-check cert mismatch: addr=%s cn=%q org=%v elapsed=%s",
			addr, leaf.Subject.CommonName, leaf.Subject.Organization, time.Since(startedAt))
		return EDRCheckResult{
			Pass:   false,
			Reason: fmt.Sprintf("unrecognized cert org: %v (cn=%q)", leaf.Subject.Organization, leaf.Subject.CommonName),
		}, nil
	}

	log.Logger.Infof("acg edr-check pass: addr=%s cn=%q org=%v elapsed=%s",
		addr, leaf.Subject.CommonName, leaf.Subject.Organization, time.Since(startedAt))
	return EDRCheckResult{Pass: true}, nil
}

// matchKnownOrg 判断证书 Organization 列表是否命中任一已知厂商（大小写不敏感）
func matchKnownOrg(orgs, known []string) bool {
	for _, o := range orgs {
		o = strings.TrimSpace(o)
		for _, k := range known {
			if strings.EqualFold(o, k) {
				return true
			}
		}
	}
	return false
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
