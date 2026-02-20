# Update hjarta-di to v0.4.1

## Overview

Update the hjarta-di dependency from v0.4.0 to v0.4.1. The main change is the CORS middleware refactoring from a struct-based API (`CORSConfig`) to functional options (`CORSOption`). The CLI keeps accepting full origin URLs with scheme (non-breaking), while internally extracting hostnames for the v0.4.1 CORS middleware. The custom `exposeHeaders` workaround is replaced by the built-in `WithExposedHeaders` option.

## Context

- Files involved: `go.mod`, `src/app/api/origins.go`, `src/app/api/origins_test.go`, `src/app/api/router.go`, `src/app/api/router_test.go`, `CLAUDE.md`
- Related patterns: hjarta-di middleware package, functional options pattern
- Dependencies: `github.com/0xalexb/hjarta-di` v0.4.0 -> v0.4.1

## v0.4.1 CORS API Summary

- `middleware.CORS(opts ...CORSOption)` replaces `middleware.CORS(middleware.CORSConfig{...})`
- Available options: `WithAllowedOrigins`, `WithAllowedMethods`, `WithAllowedHeaders`, `WithExposedHeaders`, `WithMaxAge`, `WithAllowCredentials`, `WithOriginValidators`
- Origins are now bare hostnames (e.g., `localhost`, `example.com`) instead of full URLs
- The middleware extracts the hostname from the incoming `Origin` header for matching and echoes back the full origin in `Access-Control-Allow-Origin`
- `ValidateHostname()` returns an `OriginValidator` that rejects schemes, paths, ports, wildcards, and empty strings
- Defaults: origins `["*"]`, methods `["GET", "HEAD", "POST"]`, headers `["Origin", "Accept", "Content-Type", "X-Requested-With"]`, max-age 3600

## Key Design Decision

The `--allowed-origins` CLI flag continues to accept full origin URLs with scheme (e.g., `http://localhost:3000,https://example.com`). This is a **non-breaking** CLI change. Internally, `AllowedOrigins` stores the full URLs and provides a `Hostnames()` method that extracts bare hostnames for the CORS middleware. The existing validation logic stays the same (validates scheme, no path/query/fragment).

Note: with v0.4.1's hostname-based matching, `http://localhost:3000` and `https://localhost:8080` both resolve to hostname `localhost`, so CORS will allow any origin from that host regardless of scheme/port. This is slightly broader than the current exact-URL matching but is the intended v0.4.1 behavior.

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Update go.mod dependency

**Files:**
- Modify: `go.mod`

- [x] Run `go get github.com/0xalexb/hjarta-di@v0.4.1`
- [x] Run `go mod tidy`
- [x] Run `go build ./src/...` to confirm compilation (expect errors in api package - that's OK, confirms the old CORSConfig API is removed)

### Task 2: Add Hostnames() method to AllowedOrigins

**Files:**
- Modify: `src/app/api/origins.go`
- Modify: `src/app/api/origins_test.go`

- [x] Keep all existing validation logic in `origins.go` (validateOrigin, error variables, full URL validation) - no changes to validation
- [x] Add a `Hostnames() []string` method on `AllowedOrigins` that extracts bare hostnames from each full URL using `url.Parse(entry).Hostname()` (strips port and brackets from IPv6)
- [x] Write tests for `Hostnames()`: verify `http://localhost:3000` yields `localhost`, `https://example.com` yields `example.com`, `http://127.0.0.1:8080` yields `127.0.0.1`, `http://[::1]:9090` yields `::1`
- [x] Run `go test ./src/app/api/...` - must pass before task 3

### Task 3: Update router CORS to functional options API

**Files:**
- Modify: `src/app/api/router.go`
- Modify: `src/app/api/router_test.go`

- [x] Update `NewRouter` in `router.go`: replace `middleware.CORS(middleware.CORSConfig{...})` with `middleware.CORS(middleware.WithAllowedOrigins(origins.Hostnames()...), middleware.WithAllowedMethods("GET", "POST", "DELETE", "OPTIONS"), middleware.WithAllowedHeaders("Content-Type", "Authorization", "Mcp-Session-Id", "Last-Event-ID", "Mcp-Protocol-Version"), middleware.WithExposedHeaders("Mcp-Session-Id"), middleware.WithMaxAge(corsMaxAge))`
- [x] Remove the `exposeHeaders` middleware function entirely from `router.go`
- [x] Remove `exposeHeaders` from the `router.Use(...)` middleware chain
- [x] Verify existing router tests still pass - `localhostOrigins()` keeps full URLs, the router now calls `Hostnames()` internally
- [x] Update `TestNewRouter_CORSAllowsCustomOrigin`: change configured origin to `https://example.com` (without port 443, since hostname matching is now port-agnostic) and keep request Origin header as `https://example.com` for clean matching
- [x] Run `go test ./src/app/api/...` - must pass before task 4

### Task 4: Update documentation

**Files:**
- Modify: `CLAUDE.md`

- [ ] Update CLAUDE.md: update middleware stack description - remove ExposeHeaders as separate middleware, note that CORS now uses functional options with WithExposedHeaders
- [ ] Update CLAUDE.md: update the note about hjarta-di's CORS not supporting expose headers (it now does via WithExposedHeaders)
- [ ] Update CLAUDE.md: note that AllowedOrigins stores full URLs and provides Hostnames() for CORS middleware
- [ ] Update CLAUDE.md: note that CORS matching is hostname-based (not exact URL matching)

### Task 5: Verify acceptance criteria

- [ ] Run full test suite: `go test ./src/...`
- [ ] Run linter: `golangci-lint run ./src/...`
- [ ] Verify `go build ./src/...` succeeds
- [ ] Manual test: confirm `--allowed-origins http://localhost:3000,https://example.com` is accepted (same as today)
- [ ] Manual test: confirm `--allowed-origins localhost` is rejected (missing scheme, same as today)
- [ ] Move this plan to `docs/plans/completed/`
