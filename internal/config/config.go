package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	IDPs    []IDPConfig   `yaml:"idps"`
	Logging LoggingConfig `yaml:"logging"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type IDPConfig struct {
	Name            string `yaml:"name"`
	URL             string `yaml:"url"`
	RefreshInterval int    `yaml:"refresh_interval"` // in seconds
	MaxKeys         int    `yaml:"max_keys"`         // maximum keys to maintain (default: 10)
	CacheDuration   int    `yaml:"cache_duration"`   // cache duration in seconds (default: 900)
}

// GetMaxKeys returns the max keys with a default of 10 if not set
func (c *IDPConfig) GetMaxKeys() int {
	if c.MaxKeys <= 0 {
		return 10 // Standard default
	}
	return c.MaxKeys
}

// GetCacheDuration returns the cache duration with a default of 900 seconds if not set
func (c *IDPConfig) GetCacheDuration() int {
	if c.CacheDuration <= 0 {
		return 900 // 15 minutes default
	}
	return c.CacheDuration
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
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
