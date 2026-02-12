# Version CLI Flag and MCP Server App Module

Add a `--version` / `-v` CLI flag that prints the application version and exits. Create `src/app/` package containing an MCP server struct (using `github.com/modelcontextprotocol/go-sdk`) that starts the server in a background goroutine via fx lifecycle hooks, and a DI module that wires it all together.

## Context

- Files involved: `src/main.go`, `go.mod`, new `src/app/` package
- Related patterns: hjarta-di wraps Uber fx; modules are `fx.Option` values; lifecycle hooks use `fx.Lifecycle` with `OnStart`/`OnStop`; `di.Version` is set via ldflags at build time
- Dependencies: `github.com/modelcontextprotocol/go-sdk` (v0.8.0 cached locally), `github.com/0xalexb/hjarta-di`

## Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- The version flag uses `di.Version` which is set via ldflags at build time
- The MCP server uses `mcp.NewServer` + `server.Run` with `mcp.StdioTransport` (standard MCP transport for CLI tools)
- Server start/stop is managed via fx lifecycle hooks (OnStart launches goroutine, OnStop cancels context)
- Logging uses the global `slog` package (no logger instance stored in structs)
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Task 1: Add `--version` CLI flag to main.go

**Files:**
- Modify: `src/main.go`

### Steps

- [x] In `src/main.go`, before creating the DI app, add a `flag` package check for `--version` / `-v` flag
- [x] When the flag is set, print `di.Version` to stdout and call `os.Exit(0)`
- [x] When the flag is not set, proceed with normal `di.NewApp()` + `app.Run()` flow
- [x] Write tests: create `src/main_test.go` that tests the version flag output (build and run the binary with `-version` flag, assert output contains version string)
- [x] Run `go test ./src/...` - must pass before task 2

## Task 2: Create MCP server struct in `src/app/`

**Files:**
- Create: `src/app/server.go`

### Steps

- [x] Create `src/app/server.go` with package `app`
- [x] Define a `Server` struct that holds: a `*mcp.Server` instance and a cancel function for lifecycle management (no logger field - use global `slog` functions)
- [x] Implement a `NewServer() *Server` constructor (no parameters) that creates an `mcp.Server` with `mcp.Implementation{Name: "intervals-icu-mcp", Version: di.Version}`
- [x] Implement a `Start(ctx context.Context) error` method that launches `server.Run()` in a goroutine with a cancellable context derived from the provided ctx, using `mcp.StdioTransport{}`; log with `slog.Info(...)` / `slog.Error(...)` directly
- [x] Implement a `Stop(ctx context.Context) error` method that cancels the server context to trigger graceful shutdown
- [x] Write tests: create `src/app/server_test.go` that tests NewServer creates a valid server, and tests Start/Stop lifecycle using `mcp.NewInMemoryTransports()` (or by verifying cancel behavior)
- [x] Run `go test ./src/...` - must pass before task 3

## Task 3: Create DI module in `src/app/`

**Files:**
- Create: `src/app/di.go`
- Modify: `src/main.go`

### Steps

- [x] Create `src/app/di.go` with package `app`
- [x] Define a `Module` variable of type `fx.Option` using `fx.Module("app", ...)`
- [x] In the module, use `fx.Provide(NewServer)` to register the Server constructor
- [x] In the module, use `fx.Invoke` with `fx.Lifecycle` to register OnStart/OnStop hooks that call `Server.Start` and `Server.Stop`
- [x] Modify `src/main.go` to import `src/app` and pass `app.Module` via `di.WithModules(app.Module)`
- [x] Run `go mod tidy` to pull in the go-sdk dependency
- [x] Write tests: create `src/app/di_test.go` that tests the module can be loaded into an fx test container (`fxtest.New`) and verifies the Server is provided correctly
- [x] Run `go test ./src/...` - must pass

## Verification

- [x] Manual test: run `go run ./src/... -version` and verify it prints the version
- [x] Manual test: run `go build ./src/...` and verify it compiles
- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`

## Wrap Up

- [x] Update CLAUDE.md if internal patterns changed
- [x] Move this plan to `docs/plans/completed/`
