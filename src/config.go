package src

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed config.yaml
var embeddedConfig string

//go:embed secrets.yaml
var embeddedSecrets string

// Config represents the q2git configuration
type Config struct {
	Source SourceConfig `yaml:"source"`
	Query  string       `yaml:"query"`
	Git    GitConfig    `yaml:"git"`
}

// SourceConfig represents the data source configuration
type SourceConfig struct {
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers"`
	Auth    AuthConfig        `yaml:"auth"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// GitConfig represents git repository configuration
type GitConfig struct {
	APIURL        string `yaml:"api_url"`
	Owner         string `yaml:"owner"`
	Repo          string `yaml:"repo"`
	Branch        string `yaml:"branch"`
	OutputPath    string `yaml:"output_path"`
	CommitMessage string `yaml:"commit_message"`
	Token         string `yaml:"token"`
}

// Secrets represents the secrets.yaml structure
type Secrets struct {
	Source struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"source"`
	Git struct {
		Token string `yaml:"token"`
	} `yaml:"git"`
}

// LoadConfig loads configuration from embedded config.yaml and secrets.yaml
func LoadConfig() (*Config, error) {
	// Load main configuration
	var config Config
	if err := yaml.Unmarshal([]byte(embeddedConfig), &config); err != nil {
		return nil, fmt.Errorf("failed to parse embedded config: %w", err)
	}

	// Load and merge secrets
	var secrets Secrets
	if err := yaml.Unmarshal([]byte(embeddedSecrets), &secrets); err != nil {
		return nil, fmt.Errorf("failed to parse embedded secrets: %w", err)
	}

	// Merge secrets into config
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
