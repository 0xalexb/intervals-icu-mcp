# Configure and Fix Linters

Fix all 22 golangci-lint issues using a minimal-config approach: fix everything possible in code, only exclude what truly cannot be fixed.

- Files involved: `.golangci.yml`, `src/main.go`, `src/app/server.go`, `src/app/di.go`, `src/main_test.go`, `src/app/server_test.go`
- Related patterns: existing `.golangci.yml` v2 format, `default: all` linter strategy
- Dependencies: none

## Approach

- Fix as much as possible directly in code
- Only use config exclusions or `//nolint` for cases that truly cannot be fixed
- Complete each task fully before moving to the next
- No new tests needed - these are linter/style fixes only, existing tests must continue to pass

## Issue Inventory (22 total)

| # | Linter | Count | Resolution |
|---|--------|-------|------------|
| 1 | depguard | 4 | Config: add `github.com/modelcontextprotocol` to allow lists |
| 2 | exhaustruct | 1 | Code: `//nolint:exhaustruct` - zero-value fields intentionally omitted |
| 3 | forbidigo | 1 | Code: replace `fmt.Println` with `os.Stdout.WriteString` + newline |
| 4 | gosec G204 | 2 | Config: exclude `gosec` in test files |
| 5 | ireturn | 1 | Code: `//nolint:ireturn` - returning `fx.Option` is the DI framework pattern |
| 6 | revive package-comments | 2 | Code: add package comments |
| 7 | revive unused-parameter | 1 | Code: prefix with `_` (`_ context.Context`) |
| 8 | varnamelen | 3 | Code: rename `s` to `server` in tests |
| 9 | wrapcheck | 1 | Code: wrap `ctx.Err()` with `fmt.Errorf` |
| 10 | wsl / wsl_v5 | 6 | Code: fix whitespace (add/remove blank lines) |

## Tasks

### Task 1: Fix `.golangci.yml` configuration

**Files:**
- Modify: `.golangci.yml`

- [x] Add `github.com/modelcontextprotocol` to depguard `main.allow` list (with comment `# MCP SDK`)
- [x] Add `github.com/modelcontextprotocol` to depguard `tests.allow` list (with comment `# MCP SDK`)
- [x] Add `gosec` to test file exclusions (alongside existing `err113`, `funlen`, `exhaustruct`)
- [x] Add `go.uber.org/fx/fxtest` to depguard `tests.allow` list (it's used in `di_test.go` via `fxtest.New` - currently passes because it's under `go.uber.org/fx` prefix, but being explicit is better)
- [x] Run `golangci-lint run ./src/...` - depguard and gosec issues must be resolved

### Task 2: Fix code issues in `src/app/server.go`

**Files:**
- Modify: `src/app/server.go`

- [x] Add `//nolint:exhaustruct` comment to the `&Server{...}` return in `NewServer` (zero-value fields `cancel` and `done` are intentionally nil)
- [x] Rename parameter `ctx context.Context` to `_ context.Context` in `Start()` method (unused - fx.Hook requires the signature but the server creates its own context)
- [x] Wrap `ctx.Err()` in `Stop()`: change `return ctx.Err()` to `return fmt.Errorf("waiting for server to stop: %w", ctx.Err())` and add `"fmt"` to imports
- [x] Fix wsl/wsl_v5 whitespace: remove the empty line between `case <-ctx.Done():` and `return ctx.Err()` (line 72)
- [x] Run `golangci-lint run ./src/...` - exhaustruct, revive/unused-parameter, wrapcheck, and wsl issues in server.go must be resolved

### Task 3: Fix code issues in `src/app/di.go`

**Files:**
- Modify: `src/app/di.go`

- [x] Add package comment: `// Package app provides the MCP server, its DI module, and lifecycle management.` before `package app` (only in one file per package - add it here as this is the package's entry point)
- [x] Add `//nolint:ireturn` comment to `Module()` function (returning `fx.Option` interface is the standard DI pattern)
- [x] Run `golangci-lint run ./src/...` - revive/package-comments for `app` package and ireturn must be resolved

### Task 4: Fix code issues in `src/main.go`

**Files:**
- Modify: `src/main.go`

- [x] Add package comment: `// Package main is the entry point for the intervals-icu-mcp server.` before `package main`
- [x] Replace `fmt.Println(di.Version)` with `os.Stdout.WriteString(di.Version + "\n")` and remove `"fmt"` from imports
- [x] Run `golangci-lint run ./src/...` - revive/package-comments for main and forbidigo must be resolved

### Task 5: Fix code issues in test files

**Files:**
- Modify: `src/app/server_test.go`
- Modify: `src/main_test.go`

- [x] In `src/app/server_test.go`: rename all `s := NewServer(...)` to `server := NewServer(...)` and update all references from `s.` to `server.` (3 test functions: `TestNewServer`, `TestServer_StartStop`, `TestServer_DoubleStart`)
- [x] In `src/main_test.go`: add blank line before `out, err := buildCmd.CombinedOutput()` (line 15) to satisfy wsl_v5
- [x] In `src/main_test.go`: add blank line before `output, err := cmd.Output()` (line 33) to satisfy wsl_v5
- [x] Run `golangci-lint run ./src/...` - varnamelen and wsl/wsl_v5 issues in tests must be resolved

## Verification

- [x] Run full linter: `golangci-lint run ./src/...` - must report 0 issues
- [x] Run full test suite: `go test ./src/...` - all tests must pass

## Wrap-up

- [x] Update `CLAUDE.md` if any patterns changed
