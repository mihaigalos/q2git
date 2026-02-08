package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// @Summary Execute query and optionally commit to git
// @Description Fetches data from the configured source, executes a JQ query, and optionally commits results to a git repository
// @Tags query
// @Router /api/execute [post]
// @Param commit query boolean false "Commit results to git repository"
// @Success 200 {object} object "Query results or commit success message"
// @Failure 400 {object} object "Bad request"
// @Failure 500 {object} object "Internal server error"
// @Produce json
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

	if len(config.Queries) == 0 {
		writeJSONError(w, http.StatusInternalServerError, "No queries configured", "")
		return
	}

	shouldCommit := r.URL.Query().Get("commit") == "true"
	queryName := r.URL.Query().Get("query")

	type queryResult struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Result      json.RawMessage `json:"result"`
	}

	var allResults []queryResult
	var combined []byte

	for _, q := range config.Queries {
		if queryName != "" && q.Name != queryName {
			continue
		}

		data, err := FetchData(&config.Source, q.URL)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Query '%s': failed to fetch data", q.Name), err.Error())
			return
		}

		result, err := ExecuteQuery(q.Query, data)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Query '%s' failed", q.Name), err.Error())
			return
		}

		if shouldCommit {
			var unquoted string
			if err := json.Unmarshal(result, &unquoted); err == nil {
				result = []byte(unquoted)
			}
			combined = append(combined, result...)
		} else {
			allResults = append(allResults, queryResult{
				Name:        q.Name,
				Description: q.Description,
				Result:      json.RawMessage(result),
			})
		}
	}

	if shouldCommit {
		if len(combined) == 0 {
			writeJSONError(w, http.StatusBadRequest, "No matching queries found", queryName)
			return
		}
		if err := CommitToGit(&config.Destination, &config.Settings, combined); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to commit to git", err.Error())
			return
		}
		writeCommitSuccess(w, config)
	} else {
		if len(allResults) == 1 {
			writeJSONRaw(w, allResults[0].Result)
		} else {
			out, _ := json.MarshalIndent(allResults, "", "  ")
			writeJSONRaw(w, out)
		}
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

// @Summary Root endpoint
// @Description Returns a welcome message
// @Tags general
// @Router / [get]
// @Success 200 {string} string "Welcome message"
// @Produce plain
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "Hello from TinyGo WebAssembly on wasmCloud! 🚀\n")
}

// @Summary Health check
// @Description Returns the health status of the service
// @Tags monitoring
// @Router /health [get]
// @Success 200 {object} object "Health status"
// @Produce json
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "q2git",
		"uptime":  "running",
	}
	json.NewEncoder(w).Encode(response)
}

// @Summary Service status
// @Description Returns detailed service status including version and available endpoints
// @Tags monitoring
// @Router /api/status [get]
// @Success 200 {object} object "Service status"
// @Produce json
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

// @Summary 404 handler
// @Description Returns a 404 error for unmatched routes
// @Tags general
// @Success 404 {object} object "Not found error"
// @Produce json
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
