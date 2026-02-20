# Update hjarta-di to v0.4.0 and Add HTTP Middlewares

## Overview

Update hjarta-di dependency from v0.3.0 to v0.4.0 and apply the full middleware stack from the new `listener/middleware` package to the MCP streamable HTTP transport router.

## Context

- Files involved: `go.mod`, `go.sum`, `src/app/api/router.go`, `src/app/api/router_test.go`, `CLAUDE.md`
- Related patterns: routegroup middleware via `Use()`, existing api router in `src/app/api/`
- Dependencies: `github.com/0xalexb/hjarta-di` v0.3.0 -> v0.4.0

## Design Decisions

- **Middleware location**: Applied in `src/app/api/router.go` via routegroup's `Use()` method, keeping HTTP concerns in the api package.
- **Configuration**: Hardcoded sensible defaults directly in the router. No new config struct or DI wiring needed - keeps things simple.
- **Timeout excluded**: `http.TimeoutHandler` (used internally by the Timeout middleware) cancels the request context and rejects writes after expiry. The MCP streamable transport relies on long-lived SSE connections, which would be terminated by the timeout. All other 6 middlewares are included.
- **Middleware order** (outermost to innermost): Recovery -> RequestID -> Logging -> RateLimit -> MaxRequestSize -> CORS -> Compress.

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Update hjarta-di from v0.3.0 to v0.4.0

**Files:**
- Modify: `go.mod`, `go.sum`

- [x] Run `go get github.com/0xalexb/hjarta-di@v0.4.0`
- [x] Run `go mod tidy`
- [x] Run `go build ./src/...` to verify compilation
- [x] Run `go test ./src/...` to verify existing tests still pass

### Task 2: Add middleware stack to the API router

**Files:**
- Modify: `src/app/api/router.go`

- [x] Import `github.com/0xalexb/hjarta-di/listener/middleware`
- [x] Apply middlewares to the routegroup router using `Use()` in this order:
  1. `middleware.Recovery()` - panic recovery (outermost)
  2. `middleware.RequestID()` - assigns/propagates X-Request-ID
  3. `middleware.Logging()` - structured request logging via slog
  4. `middleware.RateLimit(100, 200)` - 100 req/s, burst 200
  5. `middleware.MaxRequestSize(1048576)` - 1MB max request body
  6. `middleware.CORS(middleware.CORSConfig{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET", "POST", "DELETE", "OPTIONS"}, AllowedHeaders: []string{"Content-Type", "Authorization"}, MaxAge: 86400})` - permissive CORS for browser-based MCP clients
  7. `middleware.Compress()` - gzip compression (innermost)
- [x] Write tests for this task (see Task 3)
- [x] Run `go test ./src/...` - must pass before Task 3

### Task 3: Add tests for middleware integration

**Files:**
- Modify: `src/app/api/router_test.go`

- [x] Add test: Recovery middleware catches panics and returns 500
- [x] Add test: RequestID header (X-Request-ID) is present on responses
- [x] Add test: CORS headers set on preflight OPTIONS request to /mcp
- [x] Add test: Compress applies gzip when Accept-Encoding includes gzip and response is large enough
- [x] Add test: RateLimit returns 429 when burst is exceeded
- [x] Add test: MaxRequestSize rejects request bodies exceeding 1MB
- [x] Run `go test ./src/...`

### Task 4: Verify acceptance criteria

- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`
- [x] Verify `go build ./src/...` succeeds

### Task 5: Update documentation

- [x] Update CLAUDE.md to document the middleware stack in the Architecture section
- [x] Move this plan to `docs/plans/completed/`
