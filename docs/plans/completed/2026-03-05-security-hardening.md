# Security Hardening Based on Audit Report

## Overview

Address actionable findings from docs/security_report.md, prioritized by severity and feasibility. Excludes H1 (Login CSRF, requires session infrastructure), H2 (CORS port matching, will be fixed in hjarta-di), M4 (per-IP rate limiting, will be fixed in hjarta-di), and low-impact findings L2, L3, L6, L7.

## Context

- Files involved: `src/main.go`, `src/app/api/router.go`, `src/app/api/rest/handler.go`, `src/app/auth/jwt.go`, `src/app/auth/store.go`
- Related patterns: existing middleware stack in router.go, existing store cap pattern for clients
- Dependencies: `golang.org/x/crypto` (for HKDF key derivation)

## Deferred Findings

- **H1 (Login CSRF)**: Requires browser session binding (cookies or server-rendered consent page). Significant new infrastructure for an MCP server primarily used by programmatic clients. Recommend as a separate follow-up.
- **H2 (CORS port matching)**: Will be addressed in hjarta-di with full origin matching (scheme+host+port).
- **M4 (Per-IP rate limiting)**: Will be addressed in hjarta-di with per-IP token buckets.
- **L2 (localhost DNS)**: Minimal practical risk for a developer tool. Accepted.
- **L3 (State replay within TTL)**: 10-minute window with HMAC-signed nonce. Low risk. Accepted.
- **L6 (Test client timeout)**: Test-only code, no production impact. Accepted.
- **L7 (No JTI tracking)**: Token revocation infrastructure is complex. Short-lived tokens (1h) mitigate risk. Accepted.

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Read secrets from environment variables (H3 - HIGH)

**Files:**
- Modify: `src/main.go`

CLI flags `--github-client-secret` and `--jwt-secret` are visible in `ps aux` / `/proc/cmdline`. Add environment variable fallbacks so secrets can be provided without CLI exposure.

- [x] After `flag.Parse()`, if `githubClientSec` is empty, read from `GITHUB_CLIENT_SECRET` env var
- [x] After `flag.Parse()`, if `jwtSecret` is empty, read from `JWT_SECRET` env var
- [x] Write tests for the env var fallback logic (extract flag resolution into a testable function)
- [x] Run project test suite - must pass before task 2

### Task 2: Validate before consuming auth code (M2 - MEDIUM)

**Files:**
- Modify: `src/app/auth/store.go`
- Modify: `src/app/api/rest/handler.go`

Auth code is deleted before PKCE/client_id/redirect_uri validation. An attacker who observes a code can burn it with wrong parameters. Fix by validating binding parameters before deleting.

- [x] Add `GetAuthCode(code string, now time.Time) (*Code, error)` method to `Store` that reads without deleting, returning error if not found or expired
- [x] Add `DeleteAuthCode(code string)` method to `Store` that deletes by key
- [x] Reorder `validateAuthCodeGrant` in handler.go: call `GetAuthCode` first, validate code_verifier/client_id/redirect_uri, then call `DeleteAuthCode` only after all checks pass
- [x] Update existing tests for the new store methods; add test for the scenario where wrong code_verifier does not consume the code
- [x] Run project test suite - must pass before task 3

### Task 3: Derive separate state HMAC key (M5 - MEDIUM)

**Files:**
- Modify: `src/app/api/rest/handler.go`

JWT secret is reused for OAuth state HMAC signing. Key compromise affects both. Derive a separate state key using HKDF.

- [x] Add `golang.org/x/crypto` dependency (for `hkdf` package)
- [x] Add depguard allow entry for `golang.org/x/crypto/hkdf` in `.golangci.yml`
- [x] Add a `stateKey` field to `Handler`, derived from `jwtSecret` via `hkdf.New(sha256.New, jwtSecret, nil, []byte("oauth-state-hmac"))` during `NewHandler`
- [x] Update `signState` and `verifyState` to use `stateKey` instead of `jwtSecret`
- [x] Write tests verifying state signed with derived key validates correctly, and that JWT secret and state key produce different signatures
- [x] Run project test suite - must pass before task 4

### Task 4: Rate limit /oauth/register (M1 - MEDIUM)

**Files:**
- Modify: `src/app/api/router.go`

`POST /oauth/register` is unauthenticated and can fill 1000 client slots in seconds. Add a dedicated tight rate limiter for the registration endpoint.

- [x] Create a `registerRateLimit` middleware using hjarta-di's `middleware.RateLimit` with a low rate (e.g., 2 req/s, burst 5) applied only to the `/oauth/register` route
- [x] Wrap the register handler with this per-endpoint rate limiter in `NewRouter`
- [x] Write test verifying rate limiting kicks in for registration endpoint
- [x] Run project test suite - must pass before task 5

### Task 5: Cap auth codes and refresh tokens (M3 - MEDIUM)

**Files:**
- Modify: `src/app/auth/store.go`

Auth codes and refresh tokens have no size limit unlike clients (capped at 1000). Add caps.

- [x] Add `maxAuthCodes` and `maxRefreshTokens` constants (e.g., 10000 each)
- [x] In `SaveAuthCode`, check count and evict expired entries when cap is reached (same pattern as `SaveClient`)
- [x] In `SaveRefreshToken`, check count and evict expired entries when cap is reached
- [x] Add new error variables and return error from `SaveAuthCode` and `SaveRefreshToken`
- [x] Update handler.go callers to handle errors from `SaveAuthCode` and `SaveRefreshToken`
- [x] Write tests for cap enforcement and eviction behavior
- [x] Run project test suite - must pass before task 6

### Task 6: Sanitize GitHub error parameters (M6 - MEDIUM)

**Files:**
- Modify: `src/app/api/rest/handler.go`

GitHub's `error` and `error_description` query parameters are forwarded verbatim in callback error responses. Sanitize to prevent content injection.

- [x] Add a set of known GitHub OAuth error codes (e.g., `access_denied`, `temporarily_unavailable`)
- [x] In `validateCallbackParams`, replace unrecognized error codes with `server_error` and replace unrecognized descriptions with a generic message
- [x] Write tests for known and unknown error code sanitization
- [x] Run project test suite - must pass before task 7

### Task 7: Quick low-severity fixes (L1, L4, L5)

**Files:**
- Modify: `src/app/api/router.go` (L1)
- Modify: `src/app/auth/jwt.go` (L4)
- Modify: `src/app/auth/store.go` (L5)

- [x] L1: Remove `DELETE` from CORS `WithAllowedMethods` in router.go (MCP protocol only uses GET/POST)
- [x] L4: Add `"nbf": jwt.NewNumericDate(now)` claim in `IssueAccessToken`
- [x] L5: In `ConsumeRefreshToken`, return the same `errRefreshTokenNotFound` error for expired tokens instead of `errRefreshTokenExpired` (prevent existence leaking)
- [x] Update existing tests to verify nbf claim is present; verify uniform error for expired vs not-found refresh tokens
- [x] Run project test suite - must pass before task 8

### Task 8: Verify acceptance criteria

- [x] Manual test: start server with `--transport streamable` using env vars for secrets, verify startup succeeds
- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`

### Task 9: Update documentation

- [x] Update CLAUDE.md if internal patterns changed (new store methods, new dependency)
- [x] Move this plan to `docs/plans/completed/`
