package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"matrix/api/internal/service"
	"matrix/api/pkg/log"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// vagueLoginError 统一回显："账号或密码错误"。
// 模糊提示策略：账号不存在 / 密码错 / 已被软锁定 全部返回相同文案，
// 攻击者无法区分账号是否存在或是否被锁，符合用户拍板的"模糊提示"决策
const vagueLoginError = "账号或密码错误"

// Login POST /api/v1/auth/login
// 验证账号密码，签发 JWT 并在响应体中返回。
// 暴破防护：失败 5 次软锁定 15 分钟（联合 IP + username 计数，避免单 IP 锁掉整个账号）
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ip := c.ClientIP()
	ctx := c.Request.Context()

	// 登录前先查软锁定，被锁直接拒，不消耗 bcrypt 计算
	allowed, err := h.svc.LoginGuard.CheckAllowed(ctx, ip, req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if !allowed {
		// 模糊提示：与密码错误同文案，不泄露锁定状态
		c.JSON(http.StatusUnauthorized, gin.H{"error": vagueLoginError})
		return
	}

	resp, err := h.svc.Auth.Login(ctx, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			// 记录失败：达阈值时 LoginGuard 内部自动写锁定时间
			// guard 错误打日志但不阻塞响应（防御性，避免限流坏掉时整个登录入口挂掉）
			if gerr := h.svc.LoginGuard.RecordFailure(ctx, ip, req.Username); gerr != nil {
				log.Logger.Errorf("login guard record failure: %v", gerr)
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": vagueLoginError})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// 成功登录：清空该 (IP, username) 的累计失败 + 锁定
	_ = h.svc.LoginGuard.RecordSuccess(ctx, ip, req.Username)
	c.JSON(http.StatusOK, resp)
}

type larkExchangeRequest struct {
	Code string `json:"code" binding:"required"`
}

// LarkExchange POST /internal/auth/lark/exchange
// 仅供 Next.js 服务端调用（通过 X-Internal-Secret 鉴权）。
// 用飞书 OAuth code 换取系统 JWT。白名单制：未授权用户返回 403。
func (h *Handler) LarkExchange(c *gin.Context) {
	if c.GetHeader("X-Internal-Secret") != h.internalSecret {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var req larkExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.Auth.LarkExchange(c.Request.Context(), req.Code)
	if err != nil {
		// 不在白名单：明确返回 403，前端据此提示"无权限"
		if errors.Is(err, service.ErrNotAuthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "lark auth failed"})
		return
	}
	c.JSON(http.StatusOK, resp)
}
