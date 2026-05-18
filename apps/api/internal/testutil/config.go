// Package testutil 提供集成测试共用的配置加载工具。
// 配置文件格式为 TOML，路径为各测试包目录下的 testdata/config.toml。
// 文件不存在时返回 nil，调用方应在 nil 时调用 t.Skip() 跳过集成测试。
package testutil

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config 集成测试配置的顶层结构，对应 testdata/config.toml。
type Config struct {
	Postgres PostgresConfig `toml:"postgres"`
	Desktop  DesktopConfig  `toml:"desktop"`
	Kafka    KafkaConfig    `toml:"kafka"`
	AisACG   AisacgConfig   `toml:"aisacg"`
}

// PostgresConfig PostgreSQL 连接参数。
type PostgresConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	DBName   string `toml:"dbname"`
	SSLMode  string `toml:"sslmode"`
}

// IsEmpty 未配置 host 时视为空（未设置）。
func (c PostgresConfig) IsEmpty() bool {
	return c.Host == ""
}

// DSN 返回 pgx 格式的连接串。
func (c PostgresConfig) DSN() string {
	sslmode := c.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, sslmode,
	)
}

// DesktopConfig 亚信桌管 MySQL 连接参数。
type DesktopConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	DBName   string `toml:"dbname"`
}

// IsEmpty 未配置 host 时视为空（未设置）。
func (c DesktopConfig) IsEmpty() bool {
	return c.Host == ""
}

// DSN 返回 go-sql-driver/mysql 格式的连接串。
func (c DesktopConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true",
		c.User, c.Password, c.Host, c.Port, c.DBName,
	)
}

// KafkaConfig Kafka 连接参数。
type KafkaConfig struct {
	Brokers []string `toml:"brokers"`
}

// IsEmpty Brokers 为空时视为未配置。
func (c KafkaConfig) IsEmpty() bool {
	return len(c.Brokers) == 0
}

// AisacgConfig 亚信 ACG 设备连接参数。
type AisacgConfig struct {
	Host               string `toml:"host"`
	Username           string `toml:"username"`
	Password           string `toml:"password"`
	InsecureSkipVerify bool   `toml:"insecure_skip_verify"`
}

// IsEmpty 未配置 host 时视为空（未设置）。
func (c AisacgConfig) IsEmpty() bool {
	return c.Host == ""
}

// LoadConfig 从当前工作目录的 testdata/config.toml 加载测试配置。
// go test 运行时工作目录为各测试包目录，因此各包可维护独立的 testdata/config.toml。
// 文件不存在时返回 (nil, nil)，调用方应调用 t.Skip()。
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile("testdata/config.toml")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("testutil: parse config.toml: %w", err)
	}
	return &cfg, nil
}
