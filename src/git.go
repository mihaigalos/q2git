package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.wasmcloud.dev/component/net/wasihttp"
)

func CommitToGit(cfg *DestinationConfig, content []byte) error {
	if cfg.Token == "" {
		return fmt.Errorf("git token not configured")
	}

	baseSHA, err := getBranchRef(cfg)
	if err != nil {
		return err
	}

	treeSHA, err := getCommitTree(cfg, baseSHA)
	if err != nil {
		return err
	}

	blobSHA, err := createBlob(cfg, content)
	if err != nil {
		return err
	}

	newTreeSHA, err := createTree(cfg, treeSHA, blobSHA)
	if err != nil {
		return err
	}

	commitSHA, err := createCommit(cfg, newTreeSHA, baseSHA)
	if err != nil {
		return err
	}

	return updateBranchRef(cfg, commitSHA)
}

func getBranchRef(cfg *DestinationConfig) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/git/refs/heads/%s",
		cfg.APIURL, cfg.Owner, cfg.Repo, cfg.Branch)

	var refData struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}

	if err := githubAPIRequest("GET", url, cfg.Token, nil, &refData); err != nil {
		return "", fmt.Errorf("failed to get branch ref: %w", err)
	}

	return refData.Object.SHA, nil
}

func getCommitTree(cfg *DestinationConfig, commitSHA string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/git/commits/%s",
		cfg.APIURL, cfg.Owner, cfg.Repo, commitSHA)

	var commitData struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	}

	if err := githubAPIRequest("GET", url, cfg.Token, nil, &commitData); err != nil {
		return "", fmt.Errorf("failed to get commit tree: %w", err)
	}

	return commitData.Tree.SHA, nil
}

func createBlob(cfg *DestinationConfig, content []byte) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/git/blobs",
		cfg.APIURL, cfg.Owner, cfg.Repo)

	payload := map[string]string{
		"content":  base64.StdEncoding.EncodeToString(content),
		"encoding": "base64",
	}

	var blobData struct {
		SHA string `json:"sha"`
	}

	if err := githubAPIRequest("POST", url, cfg.Token, payload, &blobData); err != nil {
		return "", fmt.Errorf("failed to create blob: %w", err)
	}

	return blobData.SHA, nil
}

func createTree(cfg *DestinationConfig, baseTreeSHA, blobSHA string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/git/trees",
		cfg.APIURL, cfg.Owner, cfg.Repo)

	payload := map[string]interface{}{
		"base_tree": baseTreeSHA,
		"tree": []map[string]string{
			{
				"path": cfg.OutputPath,
				"mode": "100644",
				"type": "blob",
				"sha":  blobSHA,
			},
		},
	}

	var treeData struct {
		SHA string `json:"sha"`
	}

	if err := githubAPIRequest("POST", url, cfg.Token, payload, &treeData); err != nil {
		return "", fmt.Errorf("failed to create tree: %w", err)
	}

	return treeData.SHA, nil
}

func createCommit(cfg *DestinationConfig, treeSHA, parentSHA string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/git/commits",
		cfg.APIURL, cfg.Owner, cfg.Repo)

	message := strings.ReplaceAll(cfg.CommitMessage, "{{.Timestamp}}", time.Now().Format(time.RFC3339))

	payload := map[string]interface{}{
		"message": message,
		"tree":    treeSHA,
		"parents": []string{parentSHA},
	}

	var commitData struct {
		SHA string `json:"sha"`
	}

	if err := githubAPIRequest("POST", url, cfg.Token, payload, &commitData); err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}

	return commitData.SHA, nil
}

func updateBranchRef(cfg *DestinationConfig, commitSHA string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/git/refs/heads/%s",
		cfg.APIURL, cfg.Owner, cfg.Repo, cfg.Branch)

	payload := map[string]interface{}{
		"sha":   commitSHA,
		"force": false,
	}

	if err := githubAPIRequest("PATCH", url, cfg.Token, payload, nil); err != nil {
		return fmt.Errorf("failed to update branch ref: %w", err)
	}

	return nil
}

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
