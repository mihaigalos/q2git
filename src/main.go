//go:generate go run go.bytecodealliance.org/cmd/wit-bindgen-go generate --world hello --out ../gen ../wit
package main

import (
	"net/http"

	"go.wasmcloud.dev/component/net/wasihttp"
)

// Router maps paths to their handler functions
var router = map[string]http.HandlerFunc{
	"/":            HandleRoot,
	"/health":      HandleHealth,
	"/api/status":  HandleStatus,
	"/api/execute": HandleExecuteQuery,
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
	HandleNotFound(w, r)
}

// Since we don't run this program like a CLI, the `main` function is empty. Instead,
// we call the `handleRequest` function when an HTTP request is received.
func main() {}
