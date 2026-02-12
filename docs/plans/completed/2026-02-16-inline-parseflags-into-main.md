# Inline parseFlags into main

## Overview
Remove the separate parseFlags function and inline the flag parsing logic directly into main(). The cliConfig struct and parseFlags function add indirection without meaningful benefit since parseFlags is only called once from main.

## Context
- Files involved: `src/main.go`, `src/main_test.go`
- Related patterns: Standard Go flag parsing in main
- Dependencies: None

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Inline parseFlags into main and update tests

**Files:**
- Modify: `src/main.go`
- Modify: `src/main_test.go`

- [x] Remove the cliConfig struct from main.go
- [x] Remove the parseFlags function from main.go
- [x] Inline flag variable declarations and flag.Parse() directly in main()
- [x] Use local variables (showVersion, address, transportConfig, etc.) instead of the struct
- [x] Remove TestParseFlags_Defaults, TestParseFlags_StreamableTransport, and TestParseFlags_VersionFlag from main_test.go (they test the removed function)
- [x] Keep TestVersionFlag (integration test that exercises flag parsing via the compiled binary)
- [x] Run project test suite - must pass before next task

### Task 2: Verify acceptance criteria

- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`
- [x] Verify the binary still works with --version, --transport, --address, and --base-path flags
- [x] Move this plan to `docs/plans/completed/`
