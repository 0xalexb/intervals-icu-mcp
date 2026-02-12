# Return Full Build Information from Version MCP Tool

Update the `version` MCP tool to return all available build information instead of just the app version. The tool will return app version (`di.Version`), DI framework version (`di.DIVersion`), and compilation timestamp (`di.CompiledAt`) as a structured multi-line text response.

## Context

- Files involved: `src/app/server.go`, `src/app/server_test.go`, `CLAUDE.md`
- Related patterns: The version tool is registered via `mcp.AddTool` in `NewServer` with a `versionArgs` empty struct; existing test uses `mcp.NewInMemoryTransports()` with a real client; hjarta-di v0.2.1 exports three build variables: `Version` (default `"dev"`), `DIVersion` (default `"dev"`), `CompiledAt` (default `"unknown"`)
- Dependencies: no new dependencies (all three variables already exist in `github.com/0xalexb/hjarta-di`)

## Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- Return all build info as a single `TextContent` with `fmt.Sprintf` formatted multi-line string containing labeled fields (e.g., `Version: ...\nDI Version: ...\nCompiled At: ...`)
- Update the tool description to reflect that it now returns full build information
- Update `CLAUDE.md` to document the new ldflags for `DIVersion` and `CompiledAt`
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Task 1: Update the version tool to return full build information

**Files:**
- Modify: `src/app/server.go`
- Modify: `src/app/server_test.go`

### Steps

- [x] In `src/app/server.go`, update the `version` tool handler to return a formatted string containing all three build info fields: `di.Version`, `di.DIVersion`, and `di.CompiledAt` using `fmt.Sprintf("Version: %s\nDI Version: %s\nCompiled At: %s", di.Version, di.DIVersion, di.CompiledAt)`
- [x] Update the tool's `Description` from `"Returns the application version."` to `"Returns application build information including version, DI version, and compilation time."`
- [x] In `src/app/server_test.go`, update `TestVersionTool` to assert that the returned text contains all three build info values (`di.Version`, `di.DIVersion`, `di.CompiledAt`)
- [x] Run `go test ./src/...` - must pass

## Task 2: Update documentation

**Files:**
- Modify: `CLAUDE.md`

### Steps

- [x] Update the "Build with version" command in `CLAUDE.md` to document the full set of ldflags: `-X github.com/0xalexb/hjarta-di.Version=<version> -X github.com/0xalexb/hjarta-di.DIVersion=<di-version> -X github.com/0xalexb/hjarta-di.CompiledAt=<timestamp>`

## Verification

- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`

## Wrap Up

- [x] Move this plan to `docs/plans/completed/`
