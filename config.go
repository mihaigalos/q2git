package main

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"go.wasmcloud.dev/component/net/wasihttp"
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

// FetchData fetches data from the configured source
func FetchData(cfg *SourceConfig) ([]byte, error) {
	req, err := http.NewRequest(cfg.Method, cfg.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}

	// Add basic auth if configured
	if cfg.Auth.Username != "" && cfg.Auth.Password != "" {
		req.SetBasicAuth(cfg.Auth.Username, cfg.Auth.Password)
	}

	// Use wasmCloud's HTTP client which works in WASM
	client := &http.Client{
		Transport: &wasihttp.Transport{
			ConnectTimeout: 30 * time.Second,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// ExecuteQuery executes a jq query on the input data
func ExecuteQuery(query string, data []byte) ([]byte, error) {
	// Parse the jq query
	jqQuery, err := gojq.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jq query: %w", err)
	}

	// Unmarshal input data
	var input interface{}
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input JSON: %w", err)
	}

	// Execute the query
	var results []interface{}
	iter := jqQuery.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("query execution error: %w", err)
		}
		results = append(results, v)
	}

	// Marshal results back to JSON
	return json.MarshalIndent(results, "", "  ")
}

// CommitToGit commits the results to a git repository using GitHub API
func CommitToGit(cfg *GitConfig, content []byte) error {
	if cfg.Token == "" {
		return fmt.Errorf("git token not configured")
	}

	// Step 1: Get the current commit SHA of the branch
	refsURL := fmt.Sprintf("%s/repos/%s/%s/git/refs/heads/%s",
		cfg.APIURL, cfg.Owner, cfg.Repo, cfg.Branch)

	var refData struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}

	if err := githubAPIRequest("GET", refsURL, cfg.Token, nil, &refData); err != nil {
		return fmt.Errorf("failed to get branch ref: %w", err)
	}

	baseSHA := refData.Object.SHA

	// Step 2: Get the tree of the current commit
	commitURL := fmt.Sprintf("%s/repos/%s/%s/git/commits/%s",
		cfg.APIURL, cfg.Owner, cfg.Repo, baseSHA)

	var commitData struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	}

	if err := githubAPIRequest("GET", commitURL, cfg.Token, nil, &commitData); err != nil {
		return fmt.Errorf("failed to get commit tree: %w", err)
	}

	// Step 3: Create a new blob with the file content
	blobURL := fmt.Sprintf("%s/repos/%s/%s/git/blobs",
		cfg.APIURL, cfg.Owner, cfg.Repo)

	blobPayload := map[string]string{
		"content":  base64.StdEncoding.EncodeToString(content),
		"encoding": "base64",
	}

	var blobData struct {
		SHA string `json:"sha"`
	}

	if err := githubAPIRequest("POST", blobURL, cfg.Token, blobPayload, &blobData); err != nil {
		return fmt.Errorf("failed to create blob: %w", err)
	}

	// Step 4: Create a new tree with the updated file
	treeURL := fmt.Sprintf("%s/repos/%s/%s/git/trees",
		cfg.APIURL, cfg.Owner, cfg.Repo)

	treePayload := map[string]interface{}{
		"base_tree": commitData.Tree.SHA,
		"tree": []map[string]string{
			{
				"path": cfg.OutputPath,
				"mode": "100644",
				"type": "blob",
				"sha":  blobData.SHA,
			},
		},
	}

	var treeData struct {
		SHA string `json:"sha"`
	}

	if err := githubAPIRequest("POST", treeURL, cfg.Token, treePayload, &treeData); err != nil {
		return fmt.Errorf("failed to create tree: %w", err)
	}

	// Step 5: Create a new commit
	newCommitURL := fmt.Sprintf("%s/repos/%s/%s/git/commits",
		cfg.APIURL, cfg.Owner, cfg.Repo)

	commitMsg := strings.ReplaceAll(cfg.CommitMessage, "{{.Timestamp}}", time.Now().Format(time.RFC3339))

	commitPayload := map[string]interface{}{
		"message": commitMsg,
		"tree":    treeData.SHA,
		"parents": []string{baseSHA},
	}

	var newCommitData struct {
		SHA string `json:"sha"`
	}

	if err := githubAPIRequest("POST", newCommitURL, cfg.Token, commitPayload, &newCommitData); err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	// Step 6: Update the reference to point to the new commit
	updateRefURL := fmt.Sprintf("%s/repos/%s/%s/git/refs/heads/%s",
		cfg.APIURL, cfg.Owner, cfg.Repo, cfg.Branch)

	updatePayload := map[string]interface{}{
		"sha":   newCommitData.SHA,
		"force": false,
	}

	if err := githubAPIRequest("PATCH", updateRefURL, cfg.Token, updatePayload, nil); err != nil {
		return fmt.Errorf("failed to update branch ref: %w", err)
	}

	return nil
}

// githubAPIRequest makes an authenticated request to the GitHub API
func githubAPIRequest(method, url, token string, payload, response interface{}) error {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Use wasmCloud's HTTP client
	client := &http.Client{
		Transport: &wasihttp.Transport{
			ConnectTimeout: 30 * time.Second,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if response != nil {
		return json.Unmarshal(respBody, response)
	}

	return nil
}
