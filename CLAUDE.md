# Project: intervals-icu-mcp

## Project Structure

- Application source code lives under `src/`.
- Entry point: `src/main.go`.
- Go module root is at the repository root (`go.mod`).
- `src/app/` package contains the MCP server, its DI module, and lifecycle management.
- `src/app/client/` package contains the Intervals.icu HTTP client (auth, base URL, JSON requests).
- `src/app/api/` package contains HTTP routing configuration (mounts MCP handler at `/mcp`).
- `src/app/api/rest/` package contains OAuth 2.1 HTTP handlers (authorize, callback, token, register, metadata endpoints).
- `src/app/tools/` package contains MCP tool registrations (one file per tool).
- `src/app/auth/` package contains OAuth 2.1 core logic (config, JWT issuance/verification, in-memory store, authorization server metadata).
- `src/app/clients/github/` package contains the GitHub OAuth HTTP client (code exchange, user profile fetching).

## Architecture

- Dependency injection via [hjarta-di](https://github.com/0xalexb/hjarta-di), a wrapper around Uber's fx.
- Application bootstrap uses `di.NewApp()` with options like `di.WithModules()` and `di.WithLogLevel()`.
- The DI module is a package-level variable `app.Module` (not a function).
- MCP server uses [go-sdk](https://github.com/modelcontextprotocol/go-sdk) v1.2.0 with two transport modes:
  - **stdio** (default): `mcp.StdioTransport` + `server.Run` in a goroutine.
  - **streamable**: `mcp.StreamableHTTPHandler` wrapping `mcp.Server`; HTTP lifecycle managed by hjarta-di's `WithHTTPListener`.
- For streamable transport, the Server exposes a raw `http.Handler` via `Handler()` method, provided to DI with named tag `"mcp-raw"`. The `api.Module` wraps this in a router (via `routegroup`) that mounts it at `/mcp`, and provides the resulting `http.Handler` with tag `"mcp"`. hjarta-di's `WithHTTPListener("mcp", listener.WithAddress(addr))` manages the HTTP server lifecycle.
- For stdio transport, Server start/stop is managed via fx lifecycle hooks in `src/app/di.go`. For streamable transport, Start/Stop are no-ops since hjarta-di manages the HTTP lifecycle.
- `--version` / `-v` flag prints `di.Version` (set via ldflags) and exits.
- `--transport` flag selects transport mode ("stdio" or "streamable"; default: "stdio").
- `--address` flag sets the listen address for streamable transport (default: "127.0.0.1:8080").
- `--allowed-origins` flag sets comma-separated allowed CORS origins as full URLs (default: "" empty, no CORS allowed). Each entry must be a full origin URL with scheme (e.g., `http://localhost:3000`, `https://example.com`).
- `Transport` is a named string type (`Transport string`) in `src/app/transport_config.go` with constants `TransportStdio` and `TransportStreamable`; injected into DI via `fx.Supply()` from main.go.
- The MCP endpoint path `/mcp` is hardcoded in the api package router (no CLI flag).
- MCP tools live under `src/app/tools/`. Each tool exports a `ToolRegistration` constructor (e.g., `tools.NewVersionTool()`) that returns a `func(*mcp.Server)` calling `mcp.AddTool` with the correct type parameters. Tools are collected via fx group (`group:"mcp_tools"`) and injected into `NewServer` as `[]tools.ToolRegistration`.
- `src/app/client/di.go` contains `client.Module`, which provides Config (from env vars INTERVALS_API_KEY and INTERVALS_ATHLETE_ID) and Client. `app.Module` composes `client.Module` as a sub-module.
- `src/app/tools/di.go` contains `tools.Module`, which provides tool constructors via the fx group (`group:"mcp_tools"`). `app.Module` composes `tools.Module` as a sub-module.
- `src/app/api/di.go` contains `api.Module`, which provides `NewAllowedOrigins` (parses/validates `RawAllowedOrigins` into `AllowedOrigins`, returning a DI build error if any origin is malformed) and the routed `http.Handler` (tag `name:"mcp"`) by wrapping the raw MCP handler (tag `name:"mcp-raw"`) via `NewRouter`. `app.Module` composes `api.Module` as a sub-module.
- `src/app/api/origins.go` defines `RawAllowedOrigins` (named `string`, the raw CLI flag value supplied via `fx.Supply()` from main.go) and `AllowedOrigins` (named `[]string`, the parsed/validated list of full origin URLs). `NewAllowedOrigins` splits by comma, trims whitespace, drops empties, and validates each entry is a full origin URL with scheme (`http://` or `https://`), non-empty host, and no path/query/fragment. `AllowedOrigins` provides a `Hostnames()` method that extracts bare hostnames (without port) from the stored full URLs for use with the CORS middleware.
- The API router applies a middleware stack (from hjarta-di's `listener/middleware` package) via routegroup's `Use()` in this order (outermost to innermost): Recovery (panic recovery) -> RequestID (X-Request-ID propagation) -> Logging (structured request logging via slog) -> RateLimit (100 req/s, burst 200) -> MaxRequestSize (1MB body limit) -> CORS (`middleware.CORS(opts...)` with functional options from hjarta-di; performs hostname-based origin matching against `AllowedOrigins.Hostnames()`; allows GET/POST/DELETE/OPTIONS with MCP protocol headers Mcp-Session-Id, Last-Event-ID, Mcp-Protocol-Version; exposes Mcp-Session-Id via `WithExposedHeaders`; max-age 86400) -> Compress (gzip). Timeout middleware is deliberately excluded because it would terminate long-lived SSE connections used by the MCP streamable transport.
- The streamable HTTP handler is configured with a 30-minute idle session timeout (`SessionTimeout`) to prevent unbounded session accumulation from disconnected clients.
- **OAuth 2.1 authentication** is mandatory for streamable HTTP transport. The server acts as both an OAuth Authorization Server (proxying to GitHub for identity) and a Resource Server (validating JWTs on `/mcp`). Auth is not used for stdio transport.
- Auth requires three flags in streamable mode: `--github-client-id`, `--github-client-secret`, and `--auth-issuer`. The server refuses to start without them when `--transport streamable` is used.
- `--github-client-id` flag sets the GitHub OAuth app client ID (required for streamable).
- `--github-client-secret` flag sets the GitHub OAuth app client secret.
- `--allowed-users` flag sets comma-separated allowed GitHub usernames (empty = allow all authenticated users).
- `--jwt-secret` flag sets the HMAC-SHA256 signing key for JWT tokens (auto-generates a 32-byte random key if empty).
- `--auth-issuer` flag sets the issuer URL for the OAuth authorization server (required for streamable). Must be a full URL with http/https scheme.
- Auth CLI flags are supplied to DI as named types (`auth.GitHubClientID`, `auth.GitHubClientSecret`, `auth.RawAllowedUsers`, `auth.RawJWTSecret`, `auth.RawIssuer`) via `fx.Supply()` only when transport is streamable.
- `src/app/auth/config.go` defines named types for CLI flag values and validation constructors: `NewAllowedUsers` (comma-split, lowercase), `NewValidatedIssuer` (URL validation), `NewJWTSecret` (auto-generate if empty).
- `src/app/auth/store.go` implements an in-memory `Store` with `sync.RWMutex` for auth codes (one-time use with expiry), refresh tokens (rotation via consume-and-reissue), and registered clients (dynamic registration per RFC 7591).
- `src/app/auth/jwt.go` implements `IssueAccessToken` (HMAC-SHA256 JWT with iss/sub/exp/iat/jti/scope claims), `IssueRefreshToken` (random 32-byte base64url), and `NewTokenVerifier` (returns `auth.TokenVerifier` that maps JWT to `auth.TokenInfo`).
- `src/app/clients/github/github.go` implements `Client` with `ExchangeCode` (POST to GitHub token endpoint) and `GetUser` (GET GitHub user API). Methods accept plain `string` parameters (not auth named types) to avoid coupling. Base URLs are configurable for testing.
- `src/app/clients/github/testing.go` exports `NewTestClient(tokenURL, userURL string) *Client` for tests (follows `client/testing.go` pattern).
- `src/app/clients/github/di.go` contains `github.Module` (`fx.Module("github")`), which provides `Client`. `app.Module` composes it as `ghclient.Module`.
- `src/app/auth/metadata.go` implements `NewAuthorizationServerMetadata` (AS metadata with endpoints derived from issuer) and `NewProtectedResourceMetadata` (protected resource metadata pointing to the AS).
- `src/app/auth/di.go` contains `auth.Module`, which provides: AllowedUsers, validated Issuer, JWTSecret, Store, AuthorizationServerMetadata, ProtectedResourceMetadata, TokenVerifier. `app.Module` composes `auth.Module` as a sub-module. Re-exports `oauthex.ProtectedResourceMetadata` and `auth.TokenVerifier` as type aliases.
- `src/app/api/rest/handler.go` implements `Handler` with OAuth HTTP endpoints: `HandleAuthServerMetadata` (GET `/.well-known/oauth-authorization-server`), `HandleAuthorize` (GET `/oauth/authorize` - validates PKCE, HMAC-signs state, redirects to GitHub), `HandleCallback` (GET `/oauth/callback` - verifies state, exchanges GitHub code, checks allowlist, issues auth code), `HandleToken` (POST `/oauth/token` - authorization_code with PKCE and refresh_token grants), `HandleRegister` (POST `/oauth/register` - dynamic client registration). `HandlerParams` references types from `auth` and `clients/github`.
- `src/app/api/rest/di.go` contains `rest.Module` (`fx.Module("rest")`), which provides `Handler`. `app.Module` composes `rest.Module` as a sub-module.
- The API router (in streamable mode) registers OAuth discovery and flow endpoints (delegating to `rest.Handler`), and wraps `/mcp` with `auth.RequireBearerToken` middleware that validates JWT bearer tokens and sets `ResourceMetadataURL` to `/.well-known/oauth-protected-resource`.
- API tools receive `*client.Client` via DI and return JSON responses as TextContent.
- `src/app/client/testing.go` exports `NewTestClient()` for creating test clients with custom base URLs (used by tool tests with `httptest.Server`). `src/app/clients/github/testing.go` follows the same pattern.

## Build & Lint

- Build: `go build ./src/...`
- Build with version: `go build -ldflags "-X github.com/0xalexb/hjarta-di.Version=<version> -X github.com/0xalexb/hjarta-di.DIVersion=<di-version> -X github.com/0xalexb/hjarta-di.CompiledAt=<timestamp>" ./src/...`
- Test: `go test ./src/...`
- Lint: `golangci-lint run ./src/...`
- Lint config: `.golangci.yml`

## Linting Conventions

- Linter strategy: `default: all` with minimal exclusions - fix issues in code rather than suppressing them.
- depguard runs in strict mode; new dependencies must be added to the allow lists in `.golangci.yml`.
- `wsl` linter is disabled globally.
- Test files have exclusions for `err113`, `funlen`, `exhaustruct`, `gosec`, `cyclop`, `dupl`, `varnamelen`, `goconst`, `wsl_v5`, `lll`, `perfsprint`, `errorlint`, `errcheck`, `tagalign`, and `forcetypeassert`.
- `src/app/tools/` path has exclusions for `dupl` and `tagliatelle` (tool files share repetitive structure).
- `src/app/api/` path has a text-based exclusion for "avoid meaningless package names" (the `api` package name is accepted despite the revive suggestion).
- `src/app/` path has exclusions for `exhaustruct` (MCP server, DI, and client code use partial struct initialization extensively).
- `src/app/auth/` path has exclusions for `exhaustruct`, `tagliatelle`, `gosec`, `varnamelen`, `noinlineerr`, and `funcorder` (OAuth store and metadata code use partial struct initialization, external JSON tags, crypto operations, and short variable names extensively).
- `src/app/api/rest/` path has exclusions for `exhaustruct`, `tagliatelle`, `gosec`, `varnamelen`, `noinlineerr`, and `funcorder` (OAuth HTTP handlers use partial struct initialization, external JSON tags, crypto operations, and short variable names extensively).
- `src/app/clients/github/` path has exclusions for `exhaustruct`, `tagliatelle`, `gosec`, `varnamelen`, and `noinlineerr` (GitHub client uses partial struct initialization, external JSON tags, and short variable names).
- `src/app/clients/` path has a text-based exclusion for "avoid meaningless package names" (the `github` package name is intentional).
- `testpackage` linter is disabled globally; tests use internal package access (e.g., `package app` not `package app_test`).
- Use `//nolint:<linter>` comments only when the issue is inherent to the framework pattern (e.g., `ireturn` on `fx.Option` returns, `contextcheck` when fx lifecycle hooks require creating a new context, `gochecknoglobals` on `fx.Module` package variables).

## Tooling

- Go version: 1.25
- Module path: `github.com/0xalexb/intervals-icu-mcp`
- Auth dependencies: `github.com/golang-jwt/jwt/v5` (JWT issuance/verification), `github.com/gofrs/uuid/v5` (client ID generation)
