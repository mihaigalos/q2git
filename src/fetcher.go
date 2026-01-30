package src

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"go.wasmcloud.dev/component/net/wasihttp"
)

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
