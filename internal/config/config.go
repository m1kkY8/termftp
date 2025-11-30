package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultMaxPacketKB        = 1024
	defaultConcurrentRequests = 128
	defaultParallelStreams    = 4
	defaultBufferMiB          = 8
	defaultProgressInterval   = 75 * time.Millisecond
)

type Config struct {
	Host        string            `yaml:"host"`
	Port        int               `yaml:"port"`
	User        string            `yaml:"user"`
	Password    string            `yaml:"password"`
	Root        string            `yaml:"root"`
	Performance PerformanceConfig `yaml:"performance"`
	Cipher      string            `yaml:"cipher"`
}

type PerformanceConfig struct {
	MaxPacketKB        int `yaml:"maxPacketKB"`
	ConcurrentRequests int `yaml:"concurrentRequests"`
	ParallelStreams    int `yaml:"parallelStreams"`
	BufferMiB          int `yaml:"bufferMiB"`
	ProgressIntervalMs int `yaml:"progressIntervalMs"`
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
	cfg.applyDefaults()

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

func (cfg *Config) applyDefaults() {
	cfg.Performance.applyDefaults()
}

func (p *PerformanceConfig) applyDefaults() {
	if p.MaxPacketKB <= 0 {
		p.MaxPacketKB = defaultMaxPacketKB
	}
	if p.ConcurrentRequests <= 0 {
		p.ConcurrentRequests = defaultConcurrentRequests
	}
	if p.ParallelStreams <= 0 {
		p.ParallelStreams = defaultParallelStreams
	}
	if p.BufferMiB <= 0 {
		p.BufferMiB = defaultBufferMiB
	}
	if p.ProgressIntervalMs <= 0 {
		p.ProgressIntervalMs = int(defaultProgressInterval / time.Millisecond)
	}
}

func (cfg *Config) MaxPacketBytes() int {
	return clampInt(cfg.Performance.MaxPacketKB, 32, 4096) * 1024
}

func (cfg *Config) ConcurrentRequests() int {
	return clampInt(cfg.Performance.ConcurrentRequests, 16, 512)
}

func (cfg *Config) ParallelStreams() int {
	return clampInt(cfg.Performance.ParallelStreams, 1, 32)
}

func (cfg *Config) BufferSizeBytes() int {
	return clampInt(cfg.Performance.BufferMiB, 1, 64) * 1024 * 1024
}

func (cfg *Config) ProgressInterval() time.Duration {
	return time.Duration(clampInt(cfg.Performance.ProgressIntervalMs, 25, 1000)) * time.Millisecond
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (cfg *Config) SSHCiphers() []string {
	if strings.TrimSpace(cfg.Cipher) == "" {
		return nil
	}
	return []string{strings.TrimSpace(cfg.Cipher)}
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
