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
