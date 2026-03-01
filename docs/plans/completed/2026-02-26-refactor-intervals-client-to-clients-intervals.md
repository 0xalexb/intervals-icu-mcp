# Refactor Intervals.icu Client to clients/intervals

## Overview

Move the Intervals.icu HTTP client from `src/app/client/` to `src/app/clients/intervals/` to follow the established convention where all external API clients live under `src/app/clients/<service>/`. The GitHub OAuth client already follows this pattern at `src/app/clients/github/`.

## Context

- Files involved:
  - Move: `src/app/client/client.go`, `src/app/client/client_test.go`, `src/app/client/di.go`, `src/app/client/testing.go`
  - Update imports: `src/app/di.go`, `src/app/di_test.go`, all files in `src/app/tools/` (14 tool files + 2 DI files)
  - Update config: `.golangci.yml`, `CLAUDE.md`
- Related patterns: `src/app/clients/github/` package structure (package named after service, fx.Module, testing.go helper)
- Dependencies: none new

## Development Approach

- **Testing approach**: Regular (code first, then tests) - this is a pure move/rename refactoring, existing tests move as-is
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Create clients/intervals package with moved files

**Files:**
- Create: `src/app/clients/intervals/client.go` (from `src/app/client/client.go`)
- Create: `src/app/clients/intervals/client_test.go` (from `src/app/client/client_test.go`)
- Create: `src/app/clients/intervals/di.go` (from `src/app/client/di.go`)
- Create: `src/app/clients/intervals/testing.go` (from `src/app/client/testing.go`)

- [x] Create `src/app/clients/intervals/` directory
- [x] Copy `client.go` to new location, change `package client` to `package intervals`
- [x] Copy `di.go` to new location, change package to `intervals`, update fx.Module name from `"client"` to `"intervals"`
- [x] Copy `testing.go` to new location, change package to `intervals`
- [x] Copy `client_test.go` to new location, change package to `intervals`
- [x] Run `go test ./src/app/clients/intervals/...` - must pass before task 2

### Task 2: Rewire imports and remove old package

**Files:**
- Modify: `src/app/di.go` (change import from `client` to `intervals` alias)
- Modify: `src/app/di_test.go`
- Modify: all `src/app/tools/*.go` files (14+ files importing `client`)
- Delete: `src/app/client/` (entire directory)

- [x] Update `src/app/di.go`: change import from `"...src/app/client"` to `intervals "...src/app/clients/intervals"`, update `client.Module` to `intervals.Module`
- [x] Update `src/app/di_test.go`: change import and references from `client` to `intervals`
- [x] Update all tool source files (`src/app/tools/*.go`): change import and all `client.Client` references to `intervals.Client`
- [x] Update all tool test files (`src/app/tools/*_test.go`): change import and all `client.NewTestClient` references to `intervals.NewTestClient`
- [x] Delete `src/app/client/` directory
- [x] Run `go build ./src/...` to verify compilation
- [x] Run `go test ./src/...` - must pass before task 3

### Task 3: Update linter config and documentation

**Files:**
- Modify: `.golangci.yml`
- Modify: `CLAUDE.md`

- [x] Add `src/app/clients/intervals/` linter exclusion for `noinlineerr` in `.golangci.yml` (matching the pattern used by `src/app/clients/github/`)
- [x] Update all `CLAUDE.md` references from `src/app/client/` to `src/app/clients/intervals/` (package descriptions, file paths, module references)
- [x] Run `go test ./src/...` - must pass

### Task 4: Verify acceptance criteria

- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`
- [x] Verify no remaining references to old `src/app/client` import path: `grep -r "src/app/client\"" src/`
- [x] Verify `src/app/client/` directory no longer exists

### Task 5: Update documentation

- [x] Update CLAUDE.md if internal patterns changed
- [x] Move this plan to `docs/plans/completed/`
