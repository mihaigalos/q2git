package main

import (
	"fmt"

	"go.wasmcloud.dev/component/wasmcloud"
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

// LoadConfig reads Q2GIT_CONFIG (YAML) from wasmCloud named config, then overlays
// secrets from Q2GIT_GITHUB_TOKEN, Q2GIT_SOURCE_USERNAME, and Q2GIT_SOURCE_PASSWORD
// via the wasmcloud:secrets interface.
func LoadConfig() (*Config, error) {
	raw := wasmcloud.GetConfigOrDefault("Q2GIT_CONFIG", "")
	if raw == "" {
		return nil, fmt.Errorf("Q2GIT_CONFIG is not set in wasmCloud config")
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse Q2GIT_CONFIG: %w", err)
	}

	if token, err := wasmcloud.SecretGetAndReveal("Q2GIT_GITHUB_TOKEN"); err == nil {
		cfg.Destination.Token = string(token)
	} else if token := wasmcloud.GetConfigOrDefault("Q2GIT_GITHUB_TOKEN", ""); token != "" {
		cfg.Destination.Token = token
	}

	if username, err := wasmcloud.SecretGetAndReveal("Q2GIT_SOURCE_USERNAME"); err == nil {
		cfg.Source.Auth.Username = string(username)
	} else if username := wasmcloud.GetConfigOrDefault("Q2GIT_SOURCE_USERNAME", ""); username != "" {
		cfg.Source.Auth.Username = username
	}

	if password, err := wasmcloud.SecretGetAndReveal("Q2GIT_SOURCE_PASSWORD"); err == nil {
		cfg.Source.Auth.Password = string(password)
	} else if password := wasmcloud.GetConfigOrDefault("Q2GIT_SOURCE_PASSWORD", ""); password != "" {
		cfg.Source.Auth.Password = password
	}

	return &cfg, nil
}
