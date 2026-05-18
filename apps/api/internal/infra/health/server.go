// Package health 提供 /healthz HTTP 健康检查端点。
package health

import (
	"context"
	"encoding/json"
	"net/http"
)

// Server 封装 /healthz 端点的 HTTP Server。
type Server struct {
	srv *http.Server
	mux *http.ServeMux
}

// NewServer 创建 Server，监听地址为 addr（如 ":8080"）。
func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	s := &Server{
		mux: mux,
		srv: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
	mux.HandleFunc("/healthz", s.handleHealthz)
	return s
}

// Handler 返回内部 ServeMux，供 httptest.NewServer 注入（测试专用）。
func (s *Server) Handler() http.Handler {
	return s.mux
}

// ListenAndServe 启动 HTTP Server（阻塞），供 goroutine 调用。
// 正常关闭时返回 http.ErrServerClosed，调用方应过滤该错误。
func (s *Server) ListenAndServe() error {
	return s.srv.ListenAndServe()
}

// Shutdown 优雅停止（最多等待传入 ctx 超时）。
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

// Mux 返回底层 ServeMux，供外部注册额外路由（如 API 路由与 health server 共用端口）。
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}

// handleHealthz 处理 GET /healthz 请求，返回 {"status":"ok"}。
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// 写入失败时连接已断开，无法再向客户端报错；
	// 此处无 logger 依赖，忽略该错误（仅在极端情况触发）。
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
