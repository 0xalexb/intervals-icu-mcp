# Add --allowed-origins flag with DI passthrough to router

## Overview

Add a --allowed-origins CLI flag that accepts a comma-separated list of allowed CORS origins. The raw string is supplied via DI from main.go. The api package receives the raw string through DI, parses/trims/validates it, and returns an error during DI construction if any origin is malformed. The flag defaults to "localhost,127.0.0.1,::1" (matching current hardcoded behavior).

## Context

- Files involved: src/main.go, src/app/api/router.go, src/app/api/di.go, src/app/api/router_test.go, src/app/api/origins.go (new)
- Related patterns: Transport type in src/app/transport_config.go (named type + fx.Supply), fx.Annotate with ParamTags/ResultTags in api.Module
- Dependencies: none new

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Define AllowedOrigins types, parsing, and validation

**Files:**
- Create: `src/app/api/origins.go`

- [x] Create named type `RawAllowedOrigins string` - the raw comma-separated flag value, injected from main.go
- [x] Create named type `AllowedOrigins []string` - the parsed and validated list of origin hostnames
- [x] Add function `NewAllowedOrigins(raw RawAllowedOrigins) (AllowedOrigins, error)` that: splits by comma, trims whitespace from each entry, drops empty entries, validates each entry is a non-empty hostname (no scheme, no port, no path - just a hostname like "localhost" or "example.com"), returns error with descriptive message if any entry is invalid
- [x] Write tests for NewAllowedOrigins: valid single origin, valid multiple origins, whitespace trimming, empty string produces empty list, invalid entries (e.g. containing "://", "/", ":") return error
- [x] Run project test suite - must pass before task 2

### Task 2: Wire --allowed-origins flag through main.go and DI

**Files:**
- Modify: `src/main.go`
- Modify: `src/app/api/di.go`
- Modify: `src/app/api/router.go`

- [x] Add `--allowed-origins` flag in main.go (string, default "localhost,127.0.0.1,::1")
- [x] Supply `api.RawAllowedOrigins(flagValue)` via `fx.Supply()` alongside the transport value
- [x] In api.Module (di.go), add `fx.Provide(NewAllowedOrigins)` so DI calls it with RawAllowedOrigins and gets AllowedOrigins (or build error)
- [x] Update `NewRouter` signature to accept `AllowedOrigins` as a parameter
- [x] Replace `isLocalhostOrigin` check in `localhostCORS` with an origin-matching function that extracts the hostname from the Origin header and checks it against the AllowedOrigins list
- [x] Remove the now-unused `isLocalhostOrigin` function
- [x] Rename `localhostCORS` to `originsCORS` (or similar) to reflect its configurable nature
- [x] Update existing router tests to pass AllowedOrigins parameter to NewRouter (use AllowedOrigins{"localhost", "127.0.0.1", "::1"} to preserve current test behavior)
- [x] Add new tests: CORS allows a custom allowed origin (e.g. "example.com"), CORS still rejects unlisted origins
- [x] Run project test suite - must pass before task 3

### Task 3: Verify acceptance criteria

- [x] Manual test: run with --allowed-origins "localhost,127.0.0.1,::1,example.com" and verify CORS headers for http://example.com origin
- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`

### Task 4: Update documentation

- [x] Update CLAUDE.md: add --allowed-origins flag description with default value, update CORS middleware docs to reflect configurable origins, document RawAllowedOrigins/AllowedOrigins types and DI validation
- [x] Move this plan to `docs/plans/completed/`
