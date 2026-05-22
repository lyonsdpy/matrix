package auth

import "context"

// User 是从 JWT 解析出来后存入 context 的用户信息。
type User struct {
	ID       string
	Username string
	Roles    []string
}

type contextKey struct{}

// WithUser 将认证用户写入 context，供下游 middleware / resolver 读取。
func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, contextKey{}, u)
}

// FromContext 从 context 中取出当前用户，未认证时返回 nil。
func FromContext(ctx context.Context) *User {
	u, _ := ctx.Value(contextKey{}).(*User)
	return u
}
