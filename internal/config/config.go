package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Admin    AdminConfig    `yaml:"admin"`
	Install  InstallConfig  `yaml:"install"`
	Database DatabaseConfig `yaml:"database"`
}

type ServerConfig struct {
	Port    int    `yaml:"port"`
	BaseURL string `yaml:"base_url"`
}

type AdminConfig struct {
	Token string `yaml:"token"`
}

type InstallConfig struct {
	Token string `yaml:"token"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// Load reads the config from a YAML file and applies environment variable overrides.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := &Config{
		Server:   ServerConfig{Port: 8080, BaseURL: "http://localhost:8080"},
		Database: DatabaseConfig{Path: "./data/dcportal.db"},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Environment variable overrides
	if v := os.Getenv("DCPORTAL_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid DCPORTAL_PORT value %q", v)
		}
		cfg.Server.Port = port
	}
	if v := os.Getenv("DCPORTAL_BASE_URL"); v != "" {
		cfg.Server.BaseURL = v
	}
	if v := os.Getenv("DCPORTAL_ADMIN_TOKEN"); v != "" {
		cfg.Admin.Token = v
	}
	if v := os.Getenv("DCPORTAL_INSTALL_TOKEN"); v != "" {
		cfg.Install.Token = v
	}
	if v := os.Getenv("DCPORTAL_DB_PATH"); v != "" {
		cfg.Database.Path = v
	}

	if cfg.Admin.Token == "" || cfg.Admin.Token == "change-me-to-a-secure-token" {
		return nil, fmt.Errorf("admin token must be set to a secure value (set DCPORTAL_ADMIN_TOKEN env var or update config)")
	}
	if cfg.Install.Token == "" || cfg.Install.Token == "change-me-to-a-secure-token" {
		return nil, fmt.Errorf("install token must be set to a secure value (set DCPORTAL_INSTALL_TOKEN env var or update config)")
	}

	return cfg, nil
}
