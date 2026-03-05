# Update hjarta-di to v0.5.0 and Address H2/M4 Security Findings

## Overview

Update hjarta-di from v0.4.1 to v0.5.0 to use full-origin CORS matching (fixing H2 - CORS port bypass) and per-IP rate limiting (fixing M4 - global rate limit starvation). Remove the now-unnecessary `Hostnames()` method.

## Context

- Files involved: `go.mod`, `go.sum`, `src/app/api/router.go`, `src/app/api/router_test.go`, `src/app/api/origins.go`, `src/app/api/origins_test.go`, `CLAUDE.md`
- Related patterns: existing middleware stack in router.go, existing origin validation in origins.go
- Dependencies: `github.com/0xalexb/hjarta-di` v0.4.1 -> v0.5.0

## Security Findings Addressed

- **H2 (CORS Hostname Matching Strips Port)**: v0.5.0 CORS middleware supports full origin matching (scheme+host+port). Pass full origin URLs instead of stripped hostnames.
- **M4 (Global Rate Limiting Is Not Per-IP)**: v0.5.0 adds `PerIPRateLimit` middleware with sliding window per-IP tracking. Replace global `RateLimit` with `PerIPRateLimit`.

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Update hjarta-di dependency to v0.5.0

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [x] Run `go get github.com/0xalexb/hjarta-di@v0.5.0`
- [x] Run `go mod tidy`
- [x] Run project test suite to confirm no regressions from dependency update

### Task 2: Switch CORS to full origin matching (H2)

**Files:**
- Modify: `src/app/api/router.go`
- Modify: `src/app/api/origins.go`
- Modify: `src/app/api/origins_test.go`

- [ ] In `router.go`, change `middleware.WithAllowedOrigins(params.Origins.Hostnames()...)` to `middleware.WithAllowedOrigins(params.Origins...)` to pass full origin URLs directly
- [ ] Remove the `Hostnames()` method from `origins.go` (no longer used)
- [ ] Remove all `TestAllowedOrigins_Hostnames_*` tests from `origins_test.go`
- [ ] Add a test in `router_test.go` that verifies CORS rejects a request from a different port on the same hostname (e.g., allow `http://localhost:3000` but reject `http://localhost:9999`) - this is the core H2 fix
- [ ] Run project test suite - must pass before task 3

### Task 3: Switch to per-IP rate limiting (M4)

**Files:**
- Modify: `src/app/api/router.go`
- Modify: `src/app/api/router_test.go`

- [ ] Add `"time"` import to `router.go`
- [ ] Replace global `middleware.RateLimit(rateLimitRate, rateLimitBurst)` with `middleware.PerIPRateLimit(middleware.WithRateLimit(rateLimitRate, time.Second), middleware.WithBurst(rateLimitBurst))`
- [ ] Replace register endpoint `middleware.RateLimit(registerRateLimitRate, registerRateLimitBurst)` with `middleware.PerIPRateLimit(middleware.WithRateLimit(registerRateLimitRate, time.Second), middleware.WithBurst(registerRateLimitBurst))`
- [ ] Update `TestNewRouter_RateLimitExceeded` to match per-IP sliding window behavior (adjust request count to exhaust the per-IP limit: base rate + burst within the window)
- [ ] Update `TestNewRouter_RegisterRateLimitExceeded` similarly
- [ ] Run project test suite - must pass before task 4

### Task 4: Verify acceptance criteria

- [ ] Manual test: start server with `--transport streamable` and verify CORS rejects different-port origins
- [ ] Run full test suite: `go test ./src/...`
- [ ] Run linter: `golangci-lint run ./src/...`

### Task 5: Update documentation

- [ ] Update `CLAUDE.md`: update hjarta-di version, CORS description (full origin matching instead of hostname-based), rate limiting description (per-IP instead of global), remove `Hostnames()` references
- [ ] Update `docs/security_report.md`: mark H2 and M4 as resolved
- [ ] Move this plan to `docs/plans/completed/`
