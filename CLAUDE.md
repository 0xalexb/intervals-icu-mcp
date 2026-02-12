# Project: intervals-icu-mcp

## Project Structure

- Application source code lives under `src/`.
- Entry point: `src/main.go`.
- Go module root is at the repository root (`go.mod`).
- `src/app/` package contains the MCP server, its DI module, and lifecycle management.
- `src/app/client/` package contains the Intervals.icu HTTP client (auth, base URL, JSON requests).
- `src/app/tools/` package contains MCP tool registrations (one file per tool).

## Architecture

- Dependency injection via [hjarta-di](https://github.com/0xalexb/hjarta-di), a wrapper around Uber's fx.
- Application bootstrap uses `di.NewApp()` with options like `di.WithModules()` and `di.WithLogLevel()`.
- The DI module is a package-level variable `app.Module` (not a function).
- MCP server uses [go-sdk](https://github.com/modelcontextprotocol/go-sdk) (`mcp.NewServer` + `server.Run` with `mcp.StdioTransport`).
- Server start/stop is managed via fx lifecycle hooks in `src/app/di.go`.
- `--version` / `-v` flag prints `di.Version` (set via ldflags) and exits.
- MCP tools live under `src/app/tools/`. Each tool exports a `ToolRegistration` constructor (e.g., `tools.NewVersionTool()`) that returns a `func(*mcp.Server)` calling `mcp.AddTool` with the correct type parameters. Tools are collected via fx group (`group:"mcp_tools"`) and injected into `NewServer` as `[]tools.ToolRegistration`.
- `src/app/client/di.go` contains `client.Module`, which provides Config (from env vars INTERVALS_API_KEY and INTERVALS_ATHLETE_ID) and Client. `app.Module` composes `client.Module` as a sub-module.
- `src/app/tools/di.go` contains `tools.Module`, which provides tool constructors via the fx group (`group:"mcp_tools"`). `app.Module` composes `tools.Module` as a sub-module.
- API tools receive `*client.Client` via DI and return JSON responses as TextContent.
- `src/app/client/testing.go` exports `NewTestClient()` for creating test clients with custom base URLs (used by tool tests with `httptest.Server`).

## Build & Lint

- Build: `go build ./src/...`
- Build with version: `go build -ldflags "-X github.com/0xalexb/hjarta-di.Version=<version> -X github.com/0xalexb/hjarta-di.DIVersion=<di-version> -X github.com/0xalexb/hjarta-di.CompiledAt=<timestamp>" ./src/...`
- Test: `go test ./src/...`
- Lint: `golangci-lint run ./src/...`
- Lint config: `.golangci.yml`

## Linting Conventions

- Linter strategy: `default: all` with minimal exclusions - fix issues in code rather than suppressing them.
- depguard runs in strict mode; new dependencies must be added to the allow lists in `.golangci.yml`.
- `wsl` linter is disabled globally.
- Test files have exclusions for `err113`, `funlen`, `exhaustruct`, `gosec`, `cyclop`, `dupl`, `varnamelen`, `goconst`, `wsl_v5`, `lll`, `perfsprint`, `errorlint`, `errcheck`, and `tagalign`.
- `src/app/tools/` path has exclusions for `dupl` and `tagliatelle` (tool files share repetitive structure).
- `src/app/` path has exclusions for `exhaustruct` (MCP server, DI, and client code use partial struct initialization extensively).
- `testpackage` linter is disabled globally; tests use internal package access (e.g., `package app` not `package app_test`), except for `src/main_test.go` which uses `package main_test`.
- Use `//nolint:<linter>` comments only when the issue is inherent to the framework pattern (e.g., `ireturn` on `fx.Option` returns, `contextcheck` when fx lifecycle hooks require creating a new context, `gochecknoglobals` on `fx.Module` package variables).

## Tooling

- Go version: 1.25
- Module path: `github.com/0xalexb/intervals-icu-mcp`
