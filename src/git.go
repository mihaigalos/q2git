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
