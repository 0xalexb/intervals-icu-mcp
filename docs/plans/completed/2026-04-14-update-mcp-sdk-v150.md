---

# Update MCP Go-SDK from v1.2.0 to v1.5.0

## Overview

Bump the `github.com/modelcontextprotocol/go-sdk` dependency from v1.2.0 to v1.5.0, absorbing security fixes (case-sensitive JSON decoding, DNS rebinding protection, Content-Type verification), bug fixes, and stabilized client-side OAuth from intermediate releases. No source code changes are expected.

## Context

- Files involved: `go.mod`, `go.sum`, `.golangci.yml` (if depguard triggered), `CLAUDE.md`
- Related patterns: depguard strict mode in `.golangci.yml` gates all imports; transitive deps are exempt
- Dependencies: `github.com/modelcontextprotocol/go-sdk v1.5.0` (may pull in `github.com/segmentio/encoding` as indirect)
- SDK APIs verified stable across versions: `mcp.NewServer`, `mcp.NewStreamableHTTPHandler`, `mcp.StreamableHTTPOptions{SessionTimeout}`, `mcp.StdioTransport`, `mcp.AddTool`, `mcp.NewInMemoryTransports`, `mcp.NewClient`, `auth.RequireBearerToken`, `auth.RequireBearerTokenOptions{ResourceMetadataURL, Scopes}`, `auth.TokenVerifier`, `auth.TokenInfo`, `auth.ErrInvalidToken`, `auth.ProtectedResourceMetadataHandler`, `oauthex.ProtectedResourceMetadata`
- New SDK defaults that take effect automatically: DNS rebinding protection (on), `http.CrossOriginProtection` + Content-Type verification (on), tool input-validation errors returned as tool results instead of JSON-RPC errors

## Development Approach

- **Testing approach**: Regular (no new code, so no new tests needed; existing tests validate compatibility)
- Complete each task fully before moving to the next
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Bump the module dependency

**Files:**
- Modify: `go.mod`
- Regenerated: `go.sum`

- [x] Edit `go.mod` line 10: change `github.com/modelcontextprotocol/go-sdk v1.2.0` to `github.com/modelcontextprotocol/go-sdk v1.5.0`
- [x] Run `go mod tidy` to sync checksums and transitive dependencies
- [x] Verify `go.sum` was updated (new entries for v1.5.0 and any new transitive deps)

### Task 2: Build verification and source fixes (if needed)

**Files:**
- Possibly modify: any `src/` file if a breaking API change is encountered (not expected)
- Possibly modify: `.golangci.yml` if depguard flags a newly-imported package

- [x] Run `go build ./src/...` and confirm clean compilation
- [x] If build fails due to renamed/changed SDK types, fix call sites narrowly (do not restructure)
- [x] If depguard flags a new direct import, add it to the correct allow-list in `.golangci.yml`

### Task 3: Test verification

**Files:**
- No modifications expected

- [x] Run `go test ./src/...` and confirm all tests pass
- [x] Pay special attention to `src/app/server_test.go` (MCP client/server round-trip via in-memory transport)
- [x] Pay special attention to `src/app/tools/*_test.go` (tool registration + CallTool round-trip)
- [x] If any test fails due to changed SDK behavior (e.g., tool input-validation errors now returned as tool results), update test assertions to match new behavior

### Task 4: Lint verification

**Files:**
- Possibly modify: `.golangci.yml` if depguard or other linter flags new issues

- [x] Run `golangci-lint run ./src/...` and confirm clean output
- [x] If depguard flags a newly-imported package, add it to the correct allow-list in `.golangci.yml` with a one-line comment

### Task 5: Update documentation

**Files:**
- Modify: `CLAUDE.md`

- [x] Update the Architecture section: change "MCP server uses [go-sdk] v1.2.0" to "v1.5.0" (line 20)
- [x] Move `docs/2026-04-14-update-mcp-sdk-v150.md` to `docs/plans/completed/` (or create the directory if needed)
- [x] Move this plan to `docs/plans/completed/`

### Task 6: Verify acceptance criteria

- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`
