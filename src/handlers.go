package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HandleExecuteQuery endpoint - executes the configured query and commits to git
func HandleExecuteQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed. Use POST",
		})
		return
	}

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Configuration error",
			"message": err.Error(),
		})
		return
	}

	// Fetch data from source
	data, err := FetchData(&config.Source)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to fetch data",
			"message": err.Error(),
		})
		return
	}

	// Execute jq query
	results, err := ExecuteQuery(config.Query, data)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Query execution failed",
			"message": err.Error(),
		})
		return
	}

	// Check if we should commit to git (query parameter)
	shouldCommit := r.URL.Query().Get("commit") == "true"

	if shouldCommit {
		// Commit results to git
		if err := CommitToGit(&config.Git, results); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "Failed to commit to git",
				"message": err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Query executed and results committed to git",
			"repo":    fmt.Sprintf("%s/%s", config.Git.Owner, config.Git.Repo),
			"branch":  config.Git.Branch,
			"path":    config.Git.OutputPath,
		})
	} else {
		// Return results without committing
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(results)
	}
}

// HandleRoot endpoint - returns a welcome message
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "Hello from TinyGo WebAssembly on wasmCloud! 🚀\n")
}

// HandleHealth endpoint - returns application health status
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "q2git",
		"uptime":  "running",
	}
	json.NewEncoder(w).Encode(response)
}

// HandleStatus endpoint - returns detailed application status
func HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"application": "q2git",
		"version":     "1.0.0",
		"runtime":     "TinyGo + wasmCloud",
		"status":      "operational",
		"endpoints": []string{
			"/",
			"/health",
			"/api/status",
			"/api/execute",
		},
	}
	json.NewEncoder(w).Encode(response)
}

// HandleNotFound endpoint - returns 404 for unknown paths
func HandleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	response := map[string]string{
		"error":   "Not Found",
		"path":    r.URL.Path,
		"message": fmt.Sprintf("The requested path '%s' was not found", strings.TrimSpace(r.URL.Path)),
	}
	json.NewEncoder(w).Encode(response)
}

// Allow reading request body
func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	return io.ReadAll(r.Body)
}
