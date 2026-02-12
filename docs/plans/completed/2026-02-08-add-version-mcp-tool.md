# Add Version MCP Tool

Add a `version` tool to the MCP server that returns the application version string (`di.Version`) to the caller. This is the first registered tool on the server and establishes the pattern for future tool additions.

## Context

- Files involved: `src/app/server.go`, `src/app/server_test.go`
- Related patterns: MCP go-sdk uses `mcp.AddTool` generic function for tool registration with automatic schema inference; `di.Version` is set via ldflags at build time (defaults to `"dev"`); existing server tests use `mcp.NewInMemoryTransports()` with a real client for integration-style testing
- Dependencies: no new dependencies (already using `github.com/modelcontextprotocol/go-sdk` and `github.com/0xalexb/hjarta-di`)

## Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- Use the high-level `mcp.AddTool` generic function (not the low-level `Server.AddTool`) for automatic schema inference and validation
- The tool takes no input arguments (empty struct) and returns version info as `TextContent`
- Register the tool inside `NewServer` after creating `mcp.NewServer`, keeping all server setup in one place
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Task 1: Register the version tool on the MCP server

**Files:**
- Modify: `src/app/server.go`

### Steps

- [x] Define a `versionArgs` empty struct (required by the generic `mcp.AddTool` signature) in `src/app/server.go`
- [x] In `NewServer`, after `mcp.NewServer(...)`, call `mcp.AddTool` to register a tool named `"version"` with description `"Returns the application version."` and a handler that returns a `*mcp.CallToolResult` with a single `&mcp.TextContent{Text: di.Version}`
- [x] Write tests in `src/app/server_test.go`: add a `TestVersionTool` test that creates a server with in-memory transports, starts it, connects a client, calls the `"version"` tool via `session.CallTool`, and asserts the response contains `di.Version` text
- [x] Run `go test ./src/...` - must pass

## Verification

- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`

## Wrap Up

- [x] Move this plan to `docs/plans/completed/`
