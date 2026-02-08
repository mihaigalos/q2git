package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"go.wasmcloud.dev/component/net/wasihttp"
)

func FetchData(cfg *SourceConfig, url string) ([]byte, error) {
	req, err := http.NewRequest(cfg.Method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}

	if cfg.Auth.Username != "" && cfg.Auth.Password != "" {
		req.SetBasicAuth(cfg.Auth.Username, cfg.Auth.Password)
	}

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
