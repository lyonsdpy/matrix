package service

import (
	"context"
	"errors"
	"time"

	"matrix/api/internal/repository/pg_repo"
	"matrix/api/pkg/auth"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

// authUserRepo 是 AuthService 所需的 repo 接口，定义在使用方（service 包）。
type authUserRepo interface {
	FindByUsername(ctx context.Context, username string) (*pg_repo.AuthUser, error)
}

// AuthService 处理登录认证，生成无状态 JWT session。
type AuthService struct {
	repo      authUserRepo
	jwtSecret string
	expiry    time.Duration
}

func NewAuthService(repo authUserRepo, jwtSecret string, expiry time.Duration) *AuthService {
	return &AuthService{repo: repo, jwtSecret: jwtSecret, expiry: expiry}
}

// LoginResponse 是登录成功后返回给客户端的数据。
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

// Login 验证账号密码，成功后签发 JWT。
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
