# Replace Custom CORS Middleware with hjarta-di CORS

## Overview

Replace the custom `originsCORS` middleware with hjarta-di's `middleware.CORS` and switch the `--allowed-origins` flag from bare hostnames to full origin URLs (e.g., `http://localhost:3000`). This is a breaking CLI change that adopts the standard CORS origin matching model.

## Context

- Files involved:
  - `src/app/api/origins.go` - origin validation logic (rewrite)
  - `src/app/api/router.go` - middleware stack (replace custom CORS)
  - `src/main.go` - flag default and description
  - `src/app/api/origins_test.go` - validation tests (rewrite)
  - `src/app/api/router_test.go` - router/CORS tests (update)
  - `CLAUDE.md` - documentation
- Related patterns: hjarta-di `middleware.CORS(CORSConfig{...})` at `vendor/.../middleware/cors.go`
- Dependencies: hjarta-di v0.4.0 (already present)
- Key constraint: hjarta-di's CORS middleware does NOT support `Access-Control-Expose-Headers`. The MCP protocol requires exposing `Mcp-Session-Id`, so a small supplementary middleware is needed.

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Rewrite origin validation to accept full origin URLs

**Files:**
- Modify: `src/app/api/origins.go`
- Modify: `src/app/api/origins_test.go`

- [x] Replace `validateOriginHostname` with `validateOrigin` that requires a full origin URL:
  - Must have a scheme (`http://` or `https://`)
  - Must have a non-empty host
  - Must not have a path (except empty or `/`)
  - Must not have query or fragment
  - Wildcard `*` still rejected
  - Use `net/url.Parse` for validation
- [x] Update error variables: replace `errOriginHasScheme` (no longer invalid) with `errOriginMissingScheme`; keep `errOriginHasPath`, `errOriginIsWildcard`; remove `errOriginHasPort` (ports now allowed)
- [x] Update `NewAllowedOrigins` to call the new validator
- [x] Rewrite `origins_test.go`:
  - Valid: `http://localhost:3000`, `https://example.com`, `http://127.0.0.1:8080`, `http://[::1]:9090`
  - Invalid: bare hostname `localhost`, missing scheme `example.com:3000`, with path `http://example.com/path`, with query `http://example.com?q=1`, wildcard `*`
  - Keep existing structural tests (empty string, whitespace trimming, drop empty entries, invalid among valid)
- [x] Run tests: `go test ./src/app/api/...`

### Task 2: Replace custom CORS middleware with hjarta-di's middleware.CORS

**Files:**
- Modify: `src/app/api/router.go`
- Modify: `src/app/api/router_test.go`

- [x] Remove `originsCORS` function and `isAllowedOrigin` function from `router.go`
- [x] Remove `"net/url"` and `"slices"` imports (no longer needed)
- [x] Replace `originsCORS(...)` call in `NewRouter` with `middleware.CORS(middleware.CORSConfig{...})`:
  - `AllowedOrigins`: pass `[]string(origins)` directly
  - `AllowedMethods`: `[]string{"GET", "POST", "DELETE", "OPTIONS"}`
  - `AllowedHeaders`: `[]string{"Content-Type", "Authorization", "Mcp-Session-Id", "Last-Event-ID", "Mcp-Protocol-Version"}`
  - `MaxAge`: `corsMaxAge` (86400)
  - `AllowCredentials`: false
- [x] Add `exposeHeaders` middleware function that checks if `Access-Control-Allow-Origin` is set on the response writer, and if so, sets `Access-Control-Expose-Headers: Mcp-Session-Id`. Place it after CORS in the middleware stack.
- [x] Update `router_test.go`:
  - Change `localhostOrigins()` helper to return full origin URLs: `AllowedOrigins{"http://localhost:3000", "http://localhost:8080", "http://127.0.0.1:8080", "http://[::1]:9090"}`
  - Update `TestNewRouter_CORSPreflightHeaders` - origin `http://localhost:3000` should match
  - Update `TestNewRouter_ExposeHeadersMcpSessionId` - origin `http://localhost:8080` should match
  - Update `TestNewRouter_CORSRejectsNonLocalhostOrigin` - `http://evil.com` should still be rejected
  - Update `TestNewRouter_CORSAllowsLoopbackVariants` - origins must exactly match entries in `localhostOrigins()`
  - Update `TestNewRouter_CORSAllowsCustomOrigin` - change `AllowedOrigins{"example.com", "localhost"}` to `AllowedOrigins{"https://example.com:443"}` and test against `https://example.com:443`
  - Update `TestNewRouter_CORSRejectsUnlistedOrigin` - keep `http://evil.com` test with full URL origins
- [x] Run tests: `go test ./src/app/api/...`

### Task 3: Update CLI flag default and description

**Files:**
- Modify: `src/main.go`

- [x] Change `defaultAllowedOrigins` constant from `"localhost,127.0.0.1,::1"` to `""` (empty - no CORS allowed by default; secure default)
- [x] Update flag description from `"Comma-separated list of allowed CORS origin hostnames (e.g., localhost,127.0.0.1,::1,example.com)."` to `"Comma-separated list of allowed CORS origins as full URLs (e.g., http://localhost:3000,https://example.com)."`
- [x] Run tests: `go test ./src/...`

### Task 4: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

- [x] Update `--allowed-origins` flag description from bare hostnames to full origin URLs
- [x] Update description of `RawAllowedOrigins` and `AllowedOrigins` types
- [x] Update `NewAllowedOrigins` description (now validates full origin URLs, not bare hostnames)
- [x] Update CORS middleware description: now uses `middleware.CORS(CORSConfig{...})` from hjarta-di instead of custom `originsCORS`
- [x] Add note about `exposeHeaders` middleware for `Mcp-Session-Id`
- [x] Update the middleware stack order description if changed

### Task 5: Verify acceptance criteria

- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`
- [x] Verify: custom `originsCORS` and `isAllowedOrigin` functions are fully removed
- [x] Verify: `middleware.CORS` from hjarta-di is used
- [x] Verify: `Access-Control-Expose-Headers: Mcp-Session-Id` still works via supplementary middleware
- [x] Verify: `--allowed-origins` accepts full origin URLs
