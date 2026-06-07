package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"matrix/api/pkg/log"
)

// CheckEDR POST /api/v1/acg/edr-check
//
// 公开接口（认证前调用）。由 acg-checker 浏览器端调用，后端反过来探测
// 调用方 IP 的 16721 端口，判定企业 EDR 是否在跑。
//
// 不挂 JWT 中间件、不挂 RequirePermission，原因：调用发生在飞书 OAuth
// 之前，用户尚未持有任何身份。与 /api/v1/auth/login 同属"认证前公开端点"。
func (h *Handler) CheckEDR(c *gin.Context) {
	// 注意：ClientIP 受 Gin TrustedProxies 控制。生产部署若中间存在反代，
	// 需在 server 初始化处配置 trusted proxies，否则可能拿到反代 IP 而非真实 PC。
	clientIP := c.ClientIP()

	result, err := h.svc.ACG.CheckEDR(c.Request.Context(), clientIP)
	if err != nil {
		// 仅当传参非法（如 IP 为空/格式错）才会落到这里
		log.Logger.Errorf("acg edr-check error: ip=%s err=%v", clientIP, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !result.Pass {
		log.Logger.Infof("acg edr-check fail: ip=%s reason=%s", clientIP, result.Reason)
	}
	c.JSON(http.StatusOK, result)
}
