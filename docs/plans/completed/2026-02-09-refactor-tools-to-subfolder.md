# Refactor: DI module as variable, tools in subfolder, server accepts tools via DI

Refactor the MCP server setup so that:
1. The DI module is a package-level `fx.Option` variable (not a function).
2. Each MCP tool lives in its own file under `src/app/tools/`.
3. Tools are registered on the server via DI using fx group injection.

- Files involved: `src/app/di.go`, `src/app/di_test.go`, `src/app/server.go`, `src/app/server_test.go`, `src/main.go`, `CLAUDE.md`
- Related patterns: fx.Module as package variable (allowed by golang rules), fx group values for collecting dependencies, `mcp.AddTool` generic function for tool registration
- Dependencies: no new dependencies

## Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- The generic `mcp.AddTool` function requires concrete type parameters, so we cannot store typed handlers in a uniform `[]mcp.ToolHandler` slice. Instead, each tool exports a `ToolRegistration` function of type `func(*mcp.Server)` that calls `mcp.AddTool` internally with the correct type parameters.
- Tools are collected via fx group (`group:"mcp_tools"`) and injected into `NewServer` as `[]ToolRegistration`.
- The `versionArgs` struct moves into the `tools` package alongside the version tool handler.
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Task 1: Create tool registration type and version tool in subfolder

**Files:**
- Create: `src/app/tools/tools.go` — defines the `ToolRegistration` type alias (`func(*mcp.Server)`)
- Create: `src/app/tools/version.go` — the version tool (handler + args struct + exported constructor returning `ToolRegistration`)

### Steps

- [x] Create `src/app/tools/tools.go` with package declaration and `ToolRegistration` type: `type ToolRegistration func(server *mcp.Server)`
- [x] Create `src/app/tools/version.go`: move `versionArgs` struct here, create `NewVersionTool() ToolRegistration` that returns a function which calls `mcp.AddTool` on the given server
- [x] Create `src/app/tools/version_test.go`: test that `NewVersionTool()` returns a non-nil registration, and that calling it on a server registers the tool (use in-memory transports to verify the tool is callable)
- [x] Run `go test ./src/...` - must pass before task 2

## Task 2: Update server to accept tools via DI params

**Files:**
- Modify: `src/app/server.go` — `NewServer` accepts `[]ToolRegistration` and iterates to register tools
- Modify: `src/app/server_test.go` — update tests to pass tool registrations

### Steps

- [x] Add import of `tools` package in `src/app/server.go`
- [x] Change `NewServer` signature to `NewServer(transport mcp.Transport, toolRegistrations []tools.ToolRegistration) *Server`
- [x] Remove the inline `versionArgs` struct and the `mcp.AddTool` call from `NewServer`; replace with a loop: `for _, register := range toolRegistrations { register(mcpServer) }`
- [x] Update `src/app/server_test.go`: in all tests that call `NewServer`, pass `[]tools.ToolRegistration{tools.NewVersionTool()}` as the second argument
- [x] Run `go test ./src/...` - must pass before task 3

## Task 3: Convert DI module to package variable and wire tools via fx group

**Files:**
- Modify: `src/app/di.go` — change `Module()` function to `Module` variable; add fx.Provide for version tool with group tag; update NewServer param to use fx group
- Modify: `src/app/di_test.go` — update to use `Module` variable instead of `Module()`
- Modify: `src/main.go` — update to use `app.Module` instead of `app.Module()`

### Steps

- [x] In `src/app/di.go`: replace `func Module() fx.Option` with `var Module = fx.Module("app", ...)` (the nolint:ireturn comment is no longer needed)
- [x] In the module definition, add `fx.Provide(fx.Annotate(tools.NewVersionTool, fx.ResultTags(`group:"mcp_tools"`)))` to provide the version tool registration into the fx group
- [x] Update the `fx.Provide(NewServer)` to use `fx.Annotate` with `fx.ParamTags` so the `[]tools.ToolRegistration` parameter is tagged with `group:"mcp_tools"`
- [x] In `src/app/di_test.go`: change `Module()` to `Module`
- [x] In `src/main.go`: change `app.Module()` to `app.Module`
- [x] Run `go test ./src/...` - must pass before task 4

## Task 4: Update documentation

**Files:**
- Modify: `CLAUDE.md`

### Steps

- [x] Update CLAUDE.md Architecture section: document that DI module is now a package variable, tools live under `src/app/tools/`, each tool exports a `ToolRegistration` via `NewXxxTool()`, and tools are collected via fx group
- [x] Update CLAUDE.md Architecture bullet about tool registration to reflect the new pattern

## Verification

- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`
- [x] Move this plan to `docs/plans/completed/`
