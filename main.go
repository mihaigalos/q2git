//go:generate go run go.bytecodealliance.org/cmd/wit-bindgen-go generate --world hello --out gen ./wit
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.wasmcloud.dev/component/net/wasihttp"
)

// Router maps paths to their handler functions
var router = map[string]http.HandlerFunc{
	"/":            handleRoot,
	"/health":      handleHealth,
	"/api/status":  handleStatus,
	"/api/execute": handleExecuteQuery,
}

func init() {
	// Register the handleRequest function as the handler for all incoming requests.
	wasihttp.HandleFunc(handleRequest)
}

//nolint:revive
func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Look up the handler in the router
	if handler, exists := router[r.URL.Path]; exists {
		handler(w, r)
		return
	}

	// If no route matches, return 404
	handleNotFound(w, r)
}

// Root endpoint - returns a welcome message
func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "Hello from TinyGo WebAssembly on wasmCloud! 🚀\n")
}

// Health endpoint - returns application health status
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "q2git",
		"uptime":  "running",
	}
	json.NewEncoder(w).Encode(response)
}

// Status endpoint - returns detailed application status
func handleStatus(w http.ResponseWriter, r *http.Request) {
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

// ExecuteQuery endpoint - executes the configured query and commits to git
func handleExecuteQuery(w http.ResponseWriter, r *http.Request) {
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

// Allow reading request body
func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	return io.ReadAll(r.Body)
}

// NotFound endpoint - returns 404 for unknown paths
func handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	response := map[string]string{
		"error":   "Not Found",
		"path":    r.URL.Path,
		"message": fmt.Sprintf("The requested path '%s' was not found", strings.TrimSpace(r.URL.Path)),
	}
	json.NewEncoder(w).Encode(response)
}

// Since we don't run this program like a CLI, the `main` function is empty. Instead,
// we call the `handleRequest` function when an HTTP request is received.
func main() {}
