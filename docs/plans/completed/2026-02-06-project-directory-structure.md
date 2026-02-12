# Create Go Project Directory Structure

Set up the initial Go project structure with `src/main.go`, `go.mod` in root, integration with `github.com/0xalexb/hjarta-di` (a DI container wrapping Uber's fx), and a minimal golangci-lint configuration.

## Context

- Files involved: `go.mod`, `src/main.go`, `.golangci.yml`
- Related patterns: hjarta-di uses `di.NewApp()` with `di.WithModules()` and `di.WithLogLevel()` options, built on top of `go.uber.org/fx`
- Dependencies: `github.com/0xalexb/hjarta-di`
- Go version: 1.25

## Approach

- **Testing approach**: N/A - this is initial scaffolding with no business logic to test
- Complete each task fully before moving to the next

## Task 1: Create go.mod and src/main.go

**Files:**
- Create: `go.mod`
- Create: `src/main.go`

### Steps

- [x] Create `go.mod` with module path `github.com/0xalexb/intervals-icu-mcp` and Go 1.25
- [x] Create `src/main.go` with package `main`, importing `github.com/0xalexb/hjarta-di` and using `di.NewApp()` to create and start the application
- [x] Run `go mod tidy` to resolve and download dependencies

## Task 2: Create minimal golangci-lint configuration

**Files:**
- Create: `.golangci.yml`

### Steps

- [x] Create `.golangci.yml` with minimal configuration: set Go version, enable a small set of useful linters (govet, errcheck, staticcheck, unused, gosimple, ineffassign, typecheck)

## Verification

- [x] Run `go build ./src/...` to verify the project compiles
- [x] Verify directory structure matches expectations: `go.mod` and `.golangci.yml` in root, `main.go` inside `src/`
