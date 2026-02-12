# Refactor streamable transport to use hjarta-di HTTP listener

## Overview

Replace the manual http.Server management in the streamable transport with hjarta-di's WithHTTPListener. The MCP Server will expose an http.Handler, and hjarta-di will manage the HTTP server lifecycle (listen, serve, shutdown).

## Context

- Files involved: src/main.go, src/app/server.go, src/app/di.go, src/app/transport_config.go, src/app/server_test.go
- Related patterns: hjarta-di listener package (WithHTTPListener, listener.WithAddress, named http.Handler injection)
- Dependencies: github.com/0xalexb/hjarta-di v0.3.0 (already has listener support)

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Refactor Server to expose http.Handler instead of managing http.Server

**Files:**
- Modify: `src/app/server.go`
- Modify: `src/app/transport_config.go`

- [x] Remove httpSrv, listener fields from Server struct
- [x] Remove startStreamable() method (http lifecycle moves to hjarta-di)
- [x] Remove Addr() method (no longer owns the listener)
- [x] Add Handler() method that returns http.Handler (builds mux with StreamableHTTPHandler)
- [x] Keep Start/Stop only handling stdio transport; for streamable, Start/Stop become no-ops (hjarta-di manages lifecycle)
- [x] Remove Port from TransportConfig (address will be passed to WithHTTPListener); keep BasePath
- [x] Update error handling and imports
- [x] Update existing tests in src/app/server_test.go to match new behavior
- [x] Run project test suite - must pass before task 2

### Task 2: Wire hjarta-di WithHTTPListener in DI and main.go

**Files:**
- Modify: `src/app/di.go`
- Modify: `src/main.go`

- [x] In main.go, when transport is "streamable", add di.WithHTTPListener("mcp", listener.WithAddress(addr)) to di.NewApp options
- [x] In di.go, provide the http.Handler to DI with named tag "mcp" (using fx.Annotate on a constructor that calls server.Handler())
- [x] For stdio transport, do not add WithHTTPListener (no HTTP server needed)
- [x] Remove readHeaderTimeout const from server.go (hjarta-di uses its own default)
- [x] Update main.go flag handling: replace --port with --address flag (or keep --port and construct address string)
- [x] Update tests to verify the new wiring
- [x] Run project test suite - must pass before task 3

### Task 3: Clean up Server lifecycle for streamable mode

**Files:**
- Modify: `src/app/di.go`
- Modify: `src/app/server.go`

- [x] For streamable transport, Server.Start and Server.Stop should be no-ops (hjarta-di manages HTTP lifecycle)
- [x] Consider whether Server still needs fx lifecycle hooks when in streamable mode, or if only the handler provision is needed
- [x] Simplify the errAlreadyStarted / done channel logic since streamable no longer uses it
- [x] Update tests for the simplified lifecycle
- [x] Run project test suite - must pass before task 4

### Task 4: Verify acceptance criteria

- [x] Manual test: run with --transport stdio and verify MCP works
- [x] Manual test: run with --transport streamable and verify HTTP endpoint works
- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`
- [x] Verify test coverage meets 80%+

### Task 5: Update documentation

- [x] Update README.md if CLI flags changed (e.g., --port to --address)
- [x] Update CLAUDE.md to reflect that streamable transport uses hjarta-di WithHTTPListener
- [x] Move this plan to `docs/plans/completed/`
