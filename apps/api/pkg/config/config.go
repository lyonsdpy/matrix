package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"matrix/api/pkg/crypto"
	"matrix/api/pkg/log"
)

type Config struct {
	Server   Server   `yaml:"server"`
	Log      log.Conf `yaml:"log"`
	Postgres Postgres `yaml:"postgres"`
	Neo4j    Neo4j    `yaml:"neo4j"`
	JWT      JWT      `yaml:"jwt"`
	Lark     Lark     `yaml:"lark"`
}

type JWT struct {
	Secret       string `yaml:"secret"`
	ExpiryHours  int    `yaml:"expiry_hours"`
	// SecureCookie 生产环境设为 true，强制 cookie 只走 HTTPS
	SecureCookie bool   `yaml:"secure_cookie"`
}

type Server struct {
	Addr         string `yaml:"addr"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

type Postgres struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"db"`
	SSLMode  string `yaml:"sslmode"`
}

// DSN 拼装 pgx 格式连接串，供 postgres.Open 使用。
func (p Postgres) DSN() string {
	sslmode := p.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.DBName, sslmode)
}

type Neo4j struct {
	URI      string `yaml:"uri"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.decryptSecrets()
	return &cfg, nil
}

// decryptSecrets 对配置中的密码字段进行解密。
// 已加密（base64 密文）的字段自动解密；明文字段原样通过，无需任何前缀标记。
// Lark 飞书应用配置，包含基础凭据和 OAuth 参数。
type Lark struct {
	AppID          string `yaml:"app_id"`
	AppSecret      string `yaml:"app_secret"`
	OAuthRedirect  string `yaml:"oauth_redirect_uri"` // 飞书回调 URL，指向 Next.js /api/auth/callback
	InternalSecret string `yaml:"internal_secret"`    // Next.js 服务端调 Go API 内部接口时的共享密钥
}

func (c *Config) decryptSecrets() {
	c.Postgres.Password = crypto.TryDecrypt(c.Postgres.Password)
	c.Neo4j.Password = crypto.TryDecrypt(c.Neo4j.Password)
	c.JWT.Secret = crypto.TryDecrypt(c.JWT.Secret)
	c.Lark.AppSecret = crypto.TryDecrypt(c.Lark.AppSecret)
	c.Lark.InternalSecret = crypto.TryDecrypt(c.Lark.InternalSecret)
}
