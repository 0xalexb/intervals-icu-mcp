# Merge flags_test.go and main_test.go

## Overview

Merge two test files into a single main_test.go file using package main (internal access).

## Context

- Files involved: src/flags_test.go, src/main_test.go
- Related patterns: CLAUDE.md notes main_test.go uses package main_test, but merging requires a single package
- The flags_test.go uses package main for internal access to parseFlags
- The main_test.go uses package main_test for black-box binary testing
- Merging requires switching to package main since parseFlags is unexported

## Development Approach

- **Testing approach**: Regular
- Single task, straightforward merge
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Merge test files into src/main_test.go

**Files:**
- Modify: `src/main_test.go`
- Delete: `src/flags_test.go`

- [x] Rewrite src/main_test.go with package main, combining all tests from both files
- [x] Merge imports from both files (testing, context, os/exec, strings)
- [x] Include all test functions: TestParseFlags_Defaults, TestParseFlags_StreamableTransport, TestParseFlags_VersionFlag (from flags_test.go), and TestVersionFlag (from main_test.go)
- [x] Fix TestVersionFlag buildCmd.Dir from ".." to "." since package main tests run from the module root context differently - verify the correct relative path
- [x] Delete src/flags_test.go
- [x] Run go test ./src/... to verify all tests pass

### Task 2: Update CLAUDE.md

- [x] Update CLAUDE.md to remove the note about main_test.go using package main_test (since it will now use package main)

### Task 3: Verify acceptance criteria

- [x] Run full test suite: go test ./src/...
- [x] Run linter: golangci-lint run ./src/... (pre-existing v2 config issue, not related to this change)
- [x] Verify no references to flags_test.go remain
