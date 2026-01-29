//go:generate go run go.bytecodealliance.org/cmd/wit-bindgen-go generate --world hello --out gen ./wit
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.wasmcloud.dev/component/net/wasihttp"
)

// Router maps paths to their handler functions
var router = map[string]http.HandlerFunc{
	"/":           handleRoot,
	"/health":     handleHealth,
	"/api/greet":  handleGreet,
	"/api/echo":   handleEcho,
	"/api/time":   handleTime,
	"/api/status": handleStatus,
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

// Greet endpoint - greets a user by name from query parameter
func handleGreet(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"message": fmt.Sprintf("Hello, %s! Welcome to wasmCloud with TinyGo!", name),
		"name":    name,
	}
	json.NewEncoder(w).Encode(response)
}

// Echo endpoint - echoes back request information
func handleEcho(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
		"query":  r.URL.RawQuery,
		"host":   r.Host,
	}
	json.NewEncoder(w).Encode(response)
}

// Time endpoint - returns current server time
func handleTime(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"timestamp": now.Unix(),
		"iso8601":   now.Format(time.RFC3339),
		"timezone":  "UTC",
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
			"/api/greet",
			"/api/echo",
			"/api/time",
			"/api/status",
		},
	}
	json.NewEncoder(w).Encode(response)
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
