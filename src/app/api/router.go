// Package api provides HTTP routing configuration for the MCP server.
package api

import (
	"net/http"

	"github.com/go-pkgz/routegroup"
)

// NewRouter constructs an HTTP router that mounts the MCP handler at /mcp.
func NewRouter(mcpHandler http.Handler) http.Handler {
	router := routegroup.New(http.NewServeMux())
	router.Handle("/mcp", mcpHandler)
	router.Handle("/mcp/", mcpHandler)

	return router
}
