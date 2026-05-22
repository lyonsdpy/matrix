// Package crypto 提供配置文件中敏感字段（密码等）的对称加解密。
//
// 加密算法：AES-256-GCM（随机 nonce，防重放）
// 存储格式：base64(nonce + ciphertext + tag)，无需任何前缀
//
// 使用方式：
//
//	# 生成加密后的密码
//	./server -encrypt mypassword
//	# 输出：IPQYuX0q4zn5fME7PLN48a4bChntK81CZGsgmNU8ju16dstsxA==
//
//	# 直接填入 config.yaml 对应密码字段，程序启动时自动解密：
//	password: "IPQYuX0q4zn5fME7PLN48a4bChntK81CZGsgmNU8ju16dstsxA=="
//
//	# 明文密码也可直接填写，TryDecrypt 会原样通过（开发环境友好）：
//	password: "my_plain_password"
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// configKeyHex 是 AES-256 密钥的十六进制编码（32 字节 = 64 个十六进制字符）。
// 生产部署时应通过构建注入（ldflags -X）或直接替换此常量，不要提交到公共仓库。
const configKeyHex = "a3f8c2d9e6b7a4f1c8d5e2b9a6f3c0d7e4b1a8f5c2d9e6b3a0f7c4d1e8b5a2f9"

var configKey []byte

func init() {
	key, err := hex.DecodeString(configKeyHex)
	if err != nil || len(key) != 32 {
		panic(fmt.Sprintf("crypto: invalid configKeyHex (must be 64 hex chars): %v", err))
	}
	configKey = key
}

// Encrypt 对 plaintext 做 AES-256-GCM 加密，返回 base64 字符串。
// 输出可直接填入配置文件的密码字段。
func Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(configKey)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: generate nonce: %w", err)
	}
	// Seal 将 nonce 作为 dst 前缀，输出格式为 nonce+ciphertext+tag
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt 对 value 做 AES-256-GCM 严格解密，失败直接返回 error，不做明文 fallback。
// 用于 session cookie 等必须是密文的场景，防止客户端绕过加密直接填入原始 JWT。
func Decrypt(value string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("crypto: decode base64: %w", err)
	}
	block, err := aes.NewCipher(configKey)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}
	ns := gcm.NonceSize()
	if len(data) <= ns {
		return "", fmt.Errorf("crypto: ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:ns], data[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("crypto: authentication failed")
	}
	return string(plaintext), nil
}

// TryDecrypt 尝试对 value 做 AES-256-GCM 解密：
//   - 若 value 是合法 base64 且解密成功 → 返回明文
//   - 否则 → 原样返回（视为明文，兼容未加密的配置值）
//
// 适用于配置文件中的专用密码字段，调用方无需关心该字段是否已加密。
func TryDecrypt(value string) string {
	if value == "" {
		return value
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return value // 不是合法 base64，直接视为明文
	}
	block, err := aes.NewCipher(configKey)
	if err != nil {
		return value
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return value
	}
	ns := gcm.NonceSize()
	if len(data) <= ns {
		return value // 数据太短，不可能是密文
	}
	plaintext, err := gcm.Open(nil, data[:ns], data[ns:], nil)
	if err != nil {
		return value // 认证失败，视为明文（密钥不对或数据本身就是明文）
	}
	return string(plaintext)
}
