package config

import (
	"os"

	"gopkg.in/yaml.v3"
	"matrix/api/pkg/log"
)

type Config struct {
	Server Server   `yaml:"server"`
	Log    log.Conf `yaml:"log"`
}

type Server struct {
	Addr         string `yaml:"addr"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
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
	return &cfg, nil
}
