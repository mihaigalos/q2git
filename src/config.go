package main

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed config.yaml
var embeddedConfig string

//go:embed secrets.yaml
var embeddedSecrets string

type SettingsConfig struct {
	WriteMode string `yaml:"write_mode"` // "overwrite" or "append"
}

type Config struct {
	Settings    SettingsConfig    `yaml:"settings"`
	Source      SourceConfig      `yaml:"source"`
	Query       string            `yaml:"query"`
	Destination DestinationConfig `yaml:"destination"`
}

type SourceConfig struct {
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers"`
	Auth    AuthConfig        `yaml:"auth"`
}

type AuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type DestinationConfig struct {
	APIURL        string `yaml:"api_url"`
	Owner         string `yaml:"owner"`
	Repo          string `yaml:"repo"`
	Branch        string `yaml:"branch"`
	OutputPath    string `yaml:"output_path"`
	CommitMessage string `yaml:"commit_message"`
	Token         string `yaml:"token"`
}

type Secrets struct {
	Source struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"source"`
	Destination struct {
		Token string `yaml:"token"`
	} `yaml:"destination"`
}

func LoadConfig() (*Config, error) {
	var config Config
	if err := yaml.Unmarshal([]byte(embeddedConfig), &config); err != nil {
		return nil, fmt.Errorf("failed to parse embedded config: %w", err)
	}

	var secrets Secrets
	if err := yaml.Unmarshal([]byte(embeddedSecrets), &secrets); err != nil {
		return nil, fmt.Errorf("failed to parse embedded secrets: %w", err)
	}

	if secrets.Source.Username != "" {
		config.Source.Auth.Username = secrets.Source.Username
	}
	if secrets.Source.Password != "" {
		config.Source.Auth.Password = secrets.Source.Password
	}
	if secrets.Destination.Token != "" {
		config.Destination.Token = secrets.Destination.Token
	}

	return &config, nil
}
