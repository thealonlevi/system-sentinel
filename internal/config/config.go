package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SampleIntervalSec     int               `yaml:"sample_interval_sec"`
	CollectionIntervalSec int               `yaml:"collection_interval_sec"`
	LogDir                string            `yaml:"log_dir"`
	RetentionDays         int               `yaml:"retention_days"`
	Interface             string            `yaml:"interface"`
	Spikes                Spikes            `yaml:"spikes"`
	Alerts                Alerts            `yaml:"alerts"`
	Scripts               Scripts           `yaml:"scripts"`
	Env                   map[string]string `yaml:"env"`
}

type Spikes struct {
	CPU     CPUSpike     `yaml:"cpu"`
	Memory  MemorySpike  `yaml:"memory"`
	Network NetworkSpike `yaml:"network"`
}

type Alerts struct {
	CPU     CPUAlert     `yaml:"cpu"`
	Memory  MemoryAlert  `yaml:"memory"`
	Network NetworkAlert `yaml:"network"`
}

type CPUSpike struct {
	Enabled           bool    `yaml:"enabled"`
	AbsoluteThreshold float64 `yaml:"absolute_threshold"`
	RelativeThreshold float64 `yaml:"relative_threshold"`
}

type MemorySpike struct {
	Enabled           bool    `yaml:"enabled"`
	AbsoluteThreshold float64 `yaml:"absolute_threshold"`
	RelativeThreshold float64 `yaml:"relative_threshold"`
}

type NetworkSpike struct {
	Enabled           bool    `yaml:"enabled"`
	RxMbpsThreshold   float64 `yaml:"rx_mbps_threshold"`
	TxMbpsThreshold   float64 `yaml:"tx_mbps_threshold"`
	RelativeThreshold float64 `yaml:"relative_threshold"`
}

type CPUAlert struct {
	Enabled           bool    `yaml:"enabled"`
	AbsoluteThreshold float64 `yaml:"absolute_threshold"`
	RelativeThreshold float64 `yaml:"relative_threshold"`
}

type MemoryAlert struct {
	Enabled           bool    `yaml:"enabled"`
	AbsoluteThreshold float64 `yaml:"absolute_threshold"`
}

type NetworkAlert struct {
	Enabled         bool    `yaml:"enabled"`
	RxMbpsThreshold float64 `yaml:"rx_mbps_threshold"`
	TxMbpsThreshold float64 `yaml:"tx_mbps_threshold"`
}

type Scripts struct {
	Dir         string `yaml:"dir"`
	EnvFile     string `yaml:"env_file"`
	DebounceSec int    `yaml:"debounce_sec"`
	TimeoutSec  int    `yaml:"timeout_sec"`
	Enabled     bool   `yaml:"enabled"`
}

func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.SampleIntervalSec <= 0 {
		c.SampleIntervalSec = 1
	}
	if c.CollectionIntervalSec <= 0 {
		c.CollectionIntervalSec = 60
	}
	if c.LogDir == "" {
		c.LogDir = "/var/log/system-sentinel"
	}
	if c.RetentionDays <= 0 {
		c.RetentionDays = 30
	}
	if c.Interface == "" {
		c.Interface = "eth0"
	}
	if c.Scripts.Dir == "" {
		c.Scripts.Dir = "/etc/system-sentinel/sh"
	}
	if c.Scripts.EnvFile == "" {
		c.Scripts.EnvFile = "/etc/system-sentinel/.env"
	}
	if c.Scripts.DebounceSec <= 0 {
		c.Scripts.DebounceSec = 60
	}
	if c.Scripts.TimeoutSec <= 0 {
		c.Scripts.TimeoutSec = 30
	}
}

func (c *Config) validate() error {
	if c.SampleIntervalSec <= 0 {
		return fmt.Errorf("sample_interval_sec must be positive")
	}
	if c.CollectionIntervalSec <= 0 {
		return fmt.Errorf("collection_interval_sec must be positive")
	}
	if c.RetentionDays <= 0 {
		return fmt.Errorf("retention_days must be positive")
	}
	if c.Interface == "" {
		return fmt.Errorf("interface cannot be empty")
	}
	if c.Scripts.DebounceSec <= 0 {
		return fmt.Errorf("scripts.debounce_sec must be positive")
	}
	if c.Scripts.TimeoutSec <= 0 {
		return fmt.Errorf("scripts.timeout_sec must be positive")
	}
	return nil
}

func (c *Config) SampleInterval() time.Duration {
	return time.Duration(c.SampleIntervalSec) * time.Second
}

func (c *Config) CollectionInterval() time.Duration {
	return time.Duration(c.CollectionIntervalSec) * time.Second
}
