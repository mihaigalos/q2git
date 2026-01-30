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

type Config struct {
	Source SourceConfig `yaml:"source"`
	Query  string       `yaml:"query"`
	Git    GitConfig    `yaml:"git"`
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

type GitConfig struct {
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
	Git struct {
		Token string `yaml:"token"`
	} `yaml:"git"`
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
	if secrets.Git.Token != "" {
		config.Git.Token = secrets.Git.Token
	}

	return &config, nil
}
