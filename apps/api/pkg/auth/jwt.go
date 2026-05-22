package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("auth: invalid token")

// Claims 是 JWT payload 的结构，嵌入标准 RegisteredClaims。
type Claims struct {
	UserID   string   `json:"uid"`
	Username string   `json:"sub"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// GenerateToken 用 HS256 签发 JWT，expiresAt 由调用方决定，方便测试时控制过期时间。
func GenerateToken(secret, userID, username string, roles []string, expiresAt time.Time) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    "matrix-api",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("auth: sign token: %w", err)
	}
	return signed, nil
}

// ParseToken 验证签名并解析 Claims，token 过期或签名不合法均返回 ErrInvalidToken。
func ParseToken(secret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
