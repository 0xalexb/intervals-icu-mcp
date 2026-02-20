# Project: intervals-icu-mcp

## Project Structure

- Application source code lives under `src/`.
- Entry point: `src/main.go`.
- Go module root is at the repository root (`go.mod`).
- `src/app/` package contains the MCP server, its DI module, and lifecycle management.
- `src/app/client/` package contains the Intervals.icu HTTP client (auth, base URL, JSON requests).
- `src/app/api/` package contains HTTP routing configuration (mounts MCP handler at `/mcp`).
- `src/app/tools/` package contains MCP tool registrations (one file per tool).

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
- `src/app/api/origins.go` defines `RawAllowedOrigins` (named `string`, the raw CLI flag value supplied via `fx.Supply()` from main.go) and `AllowedOrigins` (named `[]string`, the parsed/validated list of full origin URLs). `NewAllowedOrigins` splits by comma, trims whitespace, drops empties, and validates each entry is a full origin URL with scheme (`http://` or `https://`), non-empty host, and no path/query/fragment.
- The API router applies a middleware stack (from hjarta-di's `listener/middleware` package and custom middlewares) via routegroup's `Use()` in this order (outermost to innermost): Recovery (panic recovery) -> RequestID (X-Request-ID propagation) -> Logging (structured request logging via slog) -> RateLimit (100 req/s, burst 200) -> MaxRequestSize (1MB body limit) -> CORS (`middleware.CORS(CORSConfig{...})` from hjarta-di; performs exact origin string matching against `AllowedOrigins`; allows GET/POST/DELETE/OPTIONS with MCP protocol headers Mcp-Session-Id, Last-Event-ID, Mcp-Protocol-Version; max-age 86400) -> ExposeHeaders (custom `exposeHeaders` middleware that sets `Access-Control-Expose-Headers: Mcp-Session-Id` when CORS has set `Access-Control-Allow-Origin`; needed because hjarta-di's CORS does not support expose headers) -> Compress (gzip). Timeout middleware is deliberately excluded because it would terminate long-lived SSE connections used by the MCP streamable transport.
- The streamable HTTP handler is configured with a 30-minute idle session timeout (`SessionTimeout`) to prevent unbounded session accumulation from disconnected clients.
- API tools receive `*client.Client` via DI and return JSON responses as TextContent.
- `src/app/client/testing.go` exports `NewTestClient()` for creating test clients with custom base URLs (used by tool tests with `httptest.Server`).

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
- Test files have exclusions for `err113`, `funlen`, `exhaustruct`, `gosec`, `cyclop`, `dupl`, `varnamelen`, `goconst`, `wsl_v5`, `lll`, `perfsprint`, `errorlint`, `errcheck`, and `tagalign`.
- `src/app/tools/` path has exclusions for `dupl` and `tagliatelle` (tool files share repetitive structure).
- `src/app/api/` path has a text-based exclusion for "avoid meaningless package names" (the `api` package name is accepted despite the revive suggestion).
- `src/app/` path has exclusions for `exhaustruct` (MCP server, DI, and client code use partial struct initialization extensively).
- `testpackage` linter is disabled globally; tests use internal package access (e.g., `package app` not `package app_test`).
- Use `//nolint:<linter>` comments only when the issue is inherent to the framework pattern (e.g., `ireturn` on `fx.Option` returns, `contextcheck` when fx lifecycle hooks require creating a new context, `gochecknoglobals` on `fx.Module` package variables).

## Tooling

- Go version: 1.25
- Module path: `github.com/0xalexb/intervals-icu-mcp`
