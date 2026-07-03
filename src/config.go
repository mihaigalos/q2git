package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type AuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type SourceConfig struct {
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers"`
	Auth    AuthConfig
}

type DestinationConfig struct {
	APIURL        string `yaml:"api_url"`
	Owner         string `yaml:"owner"`
	Repo          string `yaml:"repo"`
	Branch        string `yaml:"branch"`
	OutputPath    string `yaml:"output_path"`
	CommitMessage string `yaml:"commit_message"`
	Token         string // injected from Q2GIT_GITHUB_TOKEN, not from YAML
}

type SettingsConfig struct {
	WriteMode string `yaml:"write_mode"`
}

type QueryConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	URL         string `yaml:"url"`
	Query       string `yaml:"query"`
}

type Config struct {
	Settings    SettingsConfig    `yaml:"settings"`
	Source      SourceConfig      `yaml:"source"`
	Destination DestinationConfig `yaml:"destination"`
	Queries     []QueryConfig     `yaml:"queries"`
}

// LoadConfig reads Q2GIT_CONFIG (YAML), then overlays secrets from
// Q2GIT_GITHUB_TOKEN, Q2GIT_SOURCE_USERNAME, and Q2GIT_SOURCE_PASSWORD.
func LoadConfig() (*Config, error) {
	raw := os.Getenv("Q2GIT_CONFIG")
	if raw == "" {
		return nil, fmt.Errorf("Q2GIT_CONFIG environment variable is not set")
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse Q2GIT_CONFIG: %w", err)
	}

	if token := os.Getenv("Q2GIT_GITHUB_TOKEN"); token != "" {
		cfg.Destination.Token = token
	}

	if username := os.Getenv("Q2GIT_SOURCE_USERNAME"); username != "" {
		cfg.Source.Auth.Username = username
	}

	if password := os.Getenv("Q2GIT_SOURCE_PASSWORD"); password != "" {
		cfg.Source.Auth.Password = password
	}

	return &cfg, nil
}
