// Package api provides HTTP routing configuration for the MCP server.
package api

import (
	"net/http"
	"time"

	"github.com/0xalexb/hjarta-di/listener/middleware"
	"github.com/go-pkgz/routegroup"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"go.uber.org/fx"

	"github.com/0xalexb/intervals-icu-mcp/src/app/api/rest"
	appauth "github.com/0xalexb/intervals-icu-mcp/src/app/auth"
)

const (
	rateLimitRate      = 100
	rateLimitBurst     = 200
	maxRequestBodySize = 1048576
	corsMaxAge         = 86400

	registerRateLimitRate  = 2
	registerRateLimitBurst = 5
)

// RouterParams holds the DI-injected dependencies for the HTTP router.
type RouterParams struct {
	fx.In

	MCPHandler                  http.Handler                        `name:"mcp-raw"`
	Origins                     AllowedOrigins
	AuthHandler                 *rest.Handler
	AuthorizationServerMetadata *appauth.AuthorizationServerMetadata
	ProtectedResourceMetadata   *oauthex.ProtectedResourceMetadata
	TokenVerifier               auth.TokenVerifier
	Issuer                      appauth.Issuer
}

// NewRouter constructs an HTTP router that mounts the MCP handler at /mcp
// with the full middleware stack applied, and registers OAuth endpoints.
func NewRouter(params RouterParams) http.Handler {
	router := routegroup.New(http.NewServeMux())

	router.Use(
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.Logging(),
		middleware.PerIPRateLimit(middleware.WithRateLimit(rateLimitRate, time.Second), middleware.WithBurst(rateLimitBurst)),
		middleware.MaxRequestSize(maxRequestBodySize),
		middleware.CORS(
			middleware.WithAllowedOrigins(params.Origins...),
			middleware.WithAllowedMethods("GET", "POST", "OPTIONS"),
			middleware.WithAllowedHeaders(
				"Content-Type", "Authorization", "Mcp-Session-Id",
				"Last-Event-ID", "Mcp-Protocol-Version",
			),
			middleware.WithExposedHeaders("Mcp-Session-Id"),
			middleware.WithMaxAge(corsMaxAge),
		),
		middleware.Compress(),
	)

	resourceMetadataURL := string(params.Issuer) + "/.well-known/oauth-protected-resource"

	router.Handle("GET /.well-known/oauth-protected-resource",
		auth.ProtectedResourceMetadataHandler(params.ProtectedResourceMetadata))
	router.Handle("GET /.well-known/oauth-authorization-server",
		http.HandlerFunc(params.AuthHandler.HandleAuthServerMetadata))
	router.Handle("GET /oauth/authorize",
		http.HandlerFunc(params.AuthHandler.HandleAuthorize))
	router.Handle("GET /oauth/callback",
		http.HandlerFunc(params.AuthHandler.HandleCallback))
	router.Handle("POST /oauth/token",
		http.HandlerFunc(params.AuthHandler.HandleToken))

	registerRateLimit := middleware.PerIPRateLimit(
		middleware.WithRateLimit(registerRateLimitRate, time.Second),
		middleware.WithBurst(registerRateLimitBurst),
	)
	router.Handle("POST /oauth/register",
		registerRateLimit(http.HandlerFunc(params.AuthHandler.HandleRegister)))

	bearerMiddleware := auth.RequireBearerToken(
		params.TokenVerifier,
		&auth.RequireBearerTokenOptions{
			ResourceMetadataURL: resourceMetadataURL,
			Scopes:              []string{"mcp"},
		},
	)

	router.Handle("/mcp", bearerMiddleware(params.MCPHandler))
	router.Handle("/mcp/", bearerMiddleware(params.MCPHandler))

	return router
}
