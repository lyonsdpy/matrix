package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"matrix/api/internal/infra/lark"
	"matrix/api/internal/repository/pg_repo"
	"matrix/api/pkg/auth"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

// ErrNotAuthorized 飞书用户不在 PG users 白名单内，无权登录系统。
var ErrNotAuthorized = errors.New("user not authorized to access this system")

// authUserRepo 是 AuthService 所需的 repo 接口，定义在使用方（service 包）。
type authUserRepo interface {
	FindByUsername(ctx context.Context, username string) (*pg_repo.AuthUser, error)
	FindByLarkOpenID(ctx context.Context, openID string) (*pg_repo.AuthUser, error)
	Create(ctx context.Context, username, passwordHash string, roles []string) (*pg_repo.AuthUser, error)
	CreateLarkUser(ctx context.Context, username, larkOpenID string, roles []string) (*pg_repo.AuthUser, error)
}

// AuthService 处理登录认证，生成无状态 JWT。
// cookie 生命周期由 Next.js 服务端管理，Go API 只负责签发和验证 JWT。
type AuthService struct {
	repo      authUserRepo
	larkOAuth *lark.OAuthProvider
	jwtSecret string
	expiry    time.Duration
}

func NewAuthService(repo authUserRepo, larkOAuth *lark.OAuthProvider, jwtSecret string, expiry time.Duration) *AuthService {
	return &AuthService{repo: repo, larkOAuth: larkOAuth, jwtSecret: jwtSecret, expiry: expiry}
}

// LoginResponse 是登录成功后返回给 Next.js 的数据，由 Next.js 负责写入 cookie。
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

type UserInfo struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

// Login 验证账号密码，成功后签发 JWT（无 cookie 操作，由 Next.js 负责）。
// 故意不区分"用户不存在"和"密码错误"，防止用户名枚举攻击。
func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	if err := auth.ComparePassword(user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.issueToken(user)
}

// LarkExchange 用飞书 OAuth code 换取系统 JWT。
// 白名单制：仅当 open_id 已存在于 PG users 表（由管理员预先授权）才放行；
// 不在白名单内返回 ErrNotAuthorized——不是每个组织成员都能登录本系统。
func (s *AuthService) LarkExchange(ctx context.Context, code string) (*LoginResponse, error) {
	if s.larkOAuth == nil {
		return nil, errors.New("lark oauth not configured")
	}
	larkUser, err := s.larkOAuth.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("lark exchange: %w", err)
	}

	// 按 open_id 查白名单，不存在即拒绝（不自动创建）
	user, err := s.repo.FindByLarkOpenID(ctx, larkUser.OpenID)
	if err != nil {
		return nil, fmt.Errorf("lark exchange: lookup user: %w", err)
	}
	if user == nil {
		return nil, ErrNotAuthorized
	}

	return s.issueToken(user)
}

// issueToken 签发 JWT，复用于账号密码登录和飞书登录。
func (s *AuthService) issueToken(user *pg_repo.AuthUser) (*LoginResponse, error) {
	expiresAt := time.Now().Add(s.expiry)
	token, err := auth.GenerateToken(s.jwtSecret, user.ID, user.Username, user.Roles, expiresAt)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      UserInfo{ID: user.ID, Username: user.Username, Roles: user.Roles},
	}, nil
}
