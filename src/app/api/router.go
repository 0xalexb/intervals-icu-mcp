// Package api provides HTTP routing configuration for the MCP server.
package api

import (
	"net/http"

	"github.com/0xalexb/hjarta-di/listener/middleware"
	"github.com/go-pkgz/routegroup"
)

const (
	rateLimitRate      = 100
	rateLimitBurst     = 200
	maxRequestBodySize = 1048576
	corsMaxAge         = 86400
)

// NewRouter constructs an HTTP router that mounts the MCP handler at /mcp
// with the full middleware stack applied.
func NewRouter(mcpHandler http.Handler, origins AllowedOrigins) http.Handler {
	router := routegroup.New(http.NewServeMux())

	router.Use(
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.Logging(),
		middleware.RateLimit(rateLimitRate, rateLimitBurst),
		middleware.MaxRequestSize(maxRequestBodySize),
		middleware.CORS(
			middleware.WithAllowedOrigins(origins.Hostnames()...),
			middleware.WithAllowedMethods("GET", "POST", "DELETE", "OPTIONS"),
			middleware.WithAllowedHeaders(
				"Content-Type", "Authorization", "Mcp-Session-Id",
				"Last-Event-ID", "Mcp-Protocol-Version",
			),
			middleware.WithExposedHeaders("Mcp-Session-Id"),
			middleware.WithMaxAge(corsMaxAge),
		),
		middleware.Compress(),
	)

	router.Handle("/mcp", mcpHandler)
	router.Handle("/mcp/", mcpHandler)

	return router
}
