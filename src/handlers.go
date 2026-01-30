package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func HandleExecuteQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed. Use POST", "")
		return
	}

	config, err := LoadConfig()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Configuration error", err.Error())
		return
	}

	data, err := FetchData(&config.Source)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to fetch data", err.Error())
		return
	}

	results, err := ExecuteQuery(config.Query, data)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Query execution failed", err.Error())
		return
	}

	shouldCommit := r.URL.Query().Get("commit") == "true"

	if shouldCommit {
		if err := CommitToGit(&config.Destination, results); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to commit to git", err.Error())
			return
		}
		writeCommitSuccess(w, config)
	} else {
		writeJSONRaw(w, results)
	}
}

func writeJSONError(w http.ResponseWriter, status int, error, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	response := map[string]string{"error": error}
	if message != "" {
		response["message"] = message
	}
	json.NewEncoder(w).Encode(response)
}

func writeJSONRaw(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func writeCommitSuccess(w http.ResponseWriter, config *Config) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Query executed and results committed to git",
		"repo":    fmt.Sprintf("%s/%s", config.Destination.Owner, config.Destination.Repo),
		"branch":  config.Destination.Branch,
		"path":    config.Destination.OutputPath,
	})
}

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "Hello from TinyGo WebAssembly on wasmCloud! 🚀\n")
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "q2git",
		"uptime":  "running",
	}
	json.NewEncoder(w).Encode(response)
}

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

func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	return io.ReadAll(r.Body)
}
