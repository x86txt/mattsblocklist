// Package config provides configuration management for the tae toolkit.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	UniFi  UniFiConfig  `yaml:"unifi"`
	GitHub GitHubConfig `yaml:"github"`
}

// UniFiConfig holds UniFi controller connection settings.
type UniFiConfig struct {
	Host          string `yaml:"host"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	Site          string `yaml:"site"`
	SkipTLSVerify bool   `yaml:"skip_tls_verify"`
}

// GitHubConfig holds GitHub integration settings.
type GitHubConfig struct {
	Repo  string `yaml:"repo"`
	Token string `yaml:"token"`
}

// Load reads configuration from a YAML file and expands environment variables.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in the config
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults
	if cfg.UniFi.Site == "" {
		cfg.UniFi.Site = "default"
	}

	return &cfg, nil
}

// LoadFromEnv creates a configuration from environment variables only.
func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		UniFi: UniFiConfig{
			Host:          getEnv("UNIFI_HOST", ""),
			Username:      getEnv("UNIFI_USERNAME", ""),
			Password:      getEnv("UNIFI_PASSWORD", ""),
			Site:          getEnv("UNIFI_SITE", "default"),
			SkipTLSVerify: getEnvBool("UNIFI_SKIP_TLS_VERIFY", false),
		},
		GitHub: GitHubConfig{
			Repo:  getEnv("GITHUB_REPO", ""),
			Token: getEnv("GITHUB_TOKEN", ""),
		},
	}

	if cfg.UniFi.Host == "" {
		return nil, fmt.Errorf("UNIFI_HOST environment variable is required")
	}
	if cfg.UniFi.Username == "" {
		return nil, fmt.Errorf("UNIFI_USERNAME environment variable is required")
	}
	if cfg.UniFi.Password == "" {
		return nil, fmt.Errorf("UNIFI_PASSWORD environment variable is required")
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	val = strings.ToLower(val)
	return val == "true" || val == "1" || val == "yes"
}

