package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Root     string `yaml:"root"`
}

func LoadConfig() (*Config, error) {
	path, err := filepath.Abs("internal/config/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Host == "" {
		return errors.New("config host is required")
	}
	if cfg.User == "" {
		return errors.New("config user is required")
	}
	if cfg.Password == "" {
		return errors.New("config password is required")
	}
	if cfg.Root == "" {
		return errors.New("config root is required")
	}
	return nil
}
