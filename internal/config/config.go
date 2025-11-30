package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	path, err := resolveConfigPath()
	if err != nil {
		return nil, err
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

func resolveConfigPath() (string, error) {
	for _, candidate := range configCandidates() {
		if candidate == "" {
			continue
		}
		path := candidate
		if !filepath.IsAbs(path) {
			if abs, err := filepath.Abs(path); err == nil {
				path = abs
			}
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("config file not found in any of: %s", strings.Join(configCandidates(), ", "))
}

func configCandidates() []string {
	var paths []string
	if custom := os.Getenv("TERMFTP_CONFIG"); custom != "" {
		paths = append(paths, custom)
	}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "config.yaml"))
	}
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, "config.yaml"))
	}
	if cfgHome := os.Getenv("XDG_CONFIG_HOME"); cfgHome != "" {
		paths = append(paths, filepath.Join(cfgHome, "termftp", "config.yaml"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "termftp", "config.yaml"))
	}
	paths = append(paths, filepath.Join("internal", "config", "config.yaml"))
	return dedupStrings(paths)
}

func dedupStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result
}
