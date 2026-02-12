# Streamable HTTP Transport Support

Add streamable HTTP transport as an alternative to the default stdio transport, selectable via CLI flags (--transport, --address, --base-path).

## Context

- Files involved: go.mod, src/main.go, src/app/di.go, src/app/server.go, src/app/server_test.go, src/main_test.go
- Related patterns: fx lifecycle hooks, flag parsing in main.go, Transport interface injection via DI
- Dependencies: upgrade github.com/modelcontextprotocol/go-sdk from v0.8.0 to v1.2.0

## Implementation Notes

- **Testing approach**: Regular (code first, then tests)
- go-sdk v1.2.0 StreamableHTTPHandler does NOT use the mcp.Transport interface. Instead it wraps mcp.Server directly and acts as an http.Handler.
- For stdio: keep current approach (mcp.Transport + server.Run).
- For streamable: use StreamableHTTPHandler + http.Server with ListenAndServe.
- The Server struct needs to handle both modes: stdio (goroutine with Run) and HTTP (http.Server with ListenAndServe).
- Complete each task fully before moving to the next.
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Task 1: Upgrade go-sdk to v1.2.0

**Files:**
- Modify: `go.mod`, `go.sum`

- [x] Run `go get github.com/modelcontextprotocol/go-sdk@v1.2.0`
- [x] Run `go mod tidy`
- [x] Run `go build ./src/...` to verify no breaking changes
- [x] Run `go test ./src/...` to verify existing tests pass

## Task 2: Add CLI flags for transport selection

**Files:**
- Modify: `src/main.go`

- [x] Add `--transport` flag (values: "stdio", "streamable"; default: "stdio")
- [x] Add `--address` flag (default: ":8080", only used when transport=streamable)
- [x] Add `--base-path` flag (default: "/mcp", only used when transport=streamable)
- [x] Create a TransportConfig struct or pass individual values into the DI container
- [x] Pass transport config into `di.NewApp` so DI can provide the right transport/server setup
- [x] Write tests for flag parsing and config creation in `src/main_test.go`

## Task 3: Refactor server to support both transport modes

**Files:**
- Modify: `src/app/di.go`
- Modify: `src/app/server.go`

- [x] Define a TransportConfig type in the app package with Transport, BasePath fields
- [x] Modify `di.go`: instead of always providing StdioTransport, conditionally provide based on TransportConfig
- [x] For stdio mode: keep current behavior (mcp.Transport + server.Run in goroutine)
- [x] For streamable mode: create StreamableHTTPHandler wrapping mcp.Server, start http.Server in lifecycle hook
- [x] Modify Server struct to handle both modes (stdio uses Run, streamable uses http.Server)
- [x] Stop hook: for stdio cancel context, for streamable call http.Server.Shutdown
- [x] Write tests for both transport modes in `src/app/server_test.go`

## Verification

- [x] Manual test: run with default flags, verify stdio transport works
- [x] Manual test: run with `--transport streamable --address :9090`, verify HTTP server starts on port 9090
- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`

## Cleanup

- [x] Update CLAUDE.md if internal patterns changed (TransportConfig, new CLI flags)
- [x] Move this plan to `docs/plans/completed/`
