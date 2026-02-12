# Add API Module for HTTP Routing

## Overview

Create a new `src/app/api/` package that owns HTTP routing configuration. It uses standard `http.ServeMux` with `github.com/go-pkgz/routegroup` for route grouping. The api module constructs a router, mounts the MCP streamable handler at the hardcoded `/mcp` path, and provides the resulting `http.Handler` directly to DI with the `name:"mcp"` tag for hjarta-di's listener.

Additionally, `TransportConfig` struct is removed entirely and replaced with a `Transport` enum type (string-based). `NewServer` accepts the `Transport` enum value directly instead of the struct. The `--base-path` CLI flag is removed.

## Context

- Files involved: `src/app/server.go`, `src/app/di.go`, `src/app/transport_config.go`, `src/main.go`, new `src/app/api/` package
- Related patterns: DI module per package (`di.go`), `fx.Module` as package var, named handler injection
- Dependencies: `github.com/go-pkgz/routegroup` (new dependency to add)

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Replace TransportConfig with Transport enum

**Files:**
- Modify: `src/app/transport_config.go`
- Modify: `src/app/server.go`
- Modify: `src/app/di.go`
- Modify: `src/main.go`

- [x] In `src/app/transport_config.go`: define `Transport` as a named string type, change constants to use the new type (`TransportStdio Transport = "stdio"`, `TransportStreamable Transport = "streamable"`), remove `TransportConfig` struct entirely
- [x] In `src/app/server.go`: replace `config TransportConfig` field with `transport Transport` field (rename existing `transport mcp.Transport` field to `stdioTransport` to avoid conflict), update `NewServer` to accept `Transport` enum instead of `TransportConfig`, update all `s.config.Transport` references to use the new field directly
- [x] In `src/app/di.go`: change `fx.Supply(transportConfig)` expectation - the `Transport` value will be supplied directly from main.go
- [x] In `src/main.go`: remove `--base-path` flag and its validation, remove `TransportConfig` initialization, supply `app.Transport` value directly via `fx.Supply()`, update transport validation to use the enum type
- [x] Update existing tests that reference `TransportConfig` or `BasePath`
- [x] Run `go test ./src/...`

### Task 2: Add routegroup dependency

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `.golangci.yml`

- [x] Run `go get github.com/go-pkgz/routegroup`
- [x] Add `github.com/go-pkgz/routegroup` to both `main` and `tests` depguard allow lists in `.golangci.yml`
- [x] Run `go test ./src/...` to verify nothing breaks

### Task 3: Create the api package

**Files:**
- Create: `src/app/api/router.go`
- Create: `src/app/api/di.go`

- [x] Create `src/app/api/router.go` with a `NewRouter` constructor function that takes `http.Handler` (the raw MCP handler), constructs a `routegroup.New(http.NewServeMux())` bundle, mounts the MCP handler at `/mcp`, and returns `http.Handler`
- [x] Create `src/app/api/di.go` with `api.Module` that provides the `http.Handler` with tag `name:"mcp"` by calling `NewRouter` with the raw MCP handler (injected by `name:"mcp-raw"`)
- [x] Write tests in `src/app/api/router_test.go` verifying the MCP handler is reachable at `/mcp` and not reachable at other paths
- [x] Run `go test ./src/...`

### Task 4: Wire api module into app DI

**Files:**
- Modify: `src/app/server.go`
- Modify: `src/app/di.go`

- [x] Simplify `Server.Handler()` to return just the raw `mcp.NewStreamableHTTPHandler` without wrapping in its own `ServeMux`
- [x] Update `src/app/di.go`: change the handler provider's result tag from `name:"mcp"` to `name:"mcp-raw"` so the raw MCP handler feeds into the api module
- [x] Add `api.Module` as a sub-module of `app.Module`
- [x] Update existing server tests if they rely on the old Handler() path-mounting behavior
- [x] Run `go test ./src/...`

### Task 5: Verify acceptance criteria

- [x] Manual test: run with `--transport streamable` and verify MCP endpoint responds at `/mcp`
- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`

### Task 6: Update documentation

- [x] Update CLAUDE.md to document the api module, Transport enum type, removal of TransportConfig/--base-path, and the hardcoded `/mcp` path
- [x] Move this plan to `docs/plans/completed/`
