# Address PR Review Findings (A-1, A-2, A-3)

## Overview

Address the three actionable architecture items from the PR review in docs/pr-review-2026-03-01-v2.md. A-1 (conditional auth module loading) is implemented as a code change. A-2 and A-3 are accepted as-is with rationale.

## Context

- Files involved: `src/app/di.go`, `src/app/di_test.go`, `src/main.go`, `CLAUDE.md`
- Related patterns: fx.Module composition, conditional DI in main.go
- The PR review identified three architecture items: conditional module loading (A-1), duplicated scheme constants (A-2), and 3-value return from validateCallbackParams (A-3)

## Findings Disposition

### A-1: Conditional auth module loading - IMPLEMENT

Currently `auth.Module`, `ghclient.Module`, `rest.Module`, and `api.Module` are always loaded in `app.Module` regardless of transport. The `auth.Module` has an `fx.Invoke(registerStoreCleanup)` that eagerly starts a cleanup goroutine even for stdio transport where auth is unused. Fix: extract these modules into a `StreamableModules` grouping and load them only when transport is streamable.

### A-2: Deduplicate scheme constants - ACCEPT AS-IS

The `schemeHTTP`/`schemeHTTPS` constants are defined in both `auth/config.go` and `api/rest/handler.go`. These are trivial string constants (`"http"`, `"https"`) used in unrelated validation logic across separate packages. Extracting them to a shared package would add more complexity than the duplication warrants. The review itself notes this is minor.

### A-3: validateCallbackParams 3-value return - ACCEPT AS-IS

The function returns `(string, authorizeState, *oauthValidationError)`. In Go, 3-value returns are idiomatic and the single call site destructures into clearly named variables (`code, authState, valErr`). Wrapping in a struct would add a type for no readability gain, since the caller would then access `result.Code` and `result.State` instead of the already-descriptive local variables.

## Development Approach

- **Testing approach**: Regular (update existing tests to match new module structure)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Conditional auth module loading (A-1)

**Files:**
- Modify: `src/app/di.go`
- Modify: `src/app/di_test.go`
- Modify: `src/main.go`

**Changes to `src/app/di.go`:**
- Remove `api.Module`, `auth.Module`, `ghclient.Module`, `rest.Module` from the `Module` var
- Add a new exported `StreamableModules` package-level var that groups them: `fx.Options(api.Module, auth.Module, ghclient.Module, rest.Module)`
- Keep `intervals.Module`, `tools.Module`, server providers, and lifecycle hooks in `Module` (these are transport-agnostic)
- The `name:"mcp-raw"` provider stays in `Module` (unused but harmless in stdio mode)

**Changes to `src/main.go`:**
- Move `fx.Supply(api.RawAllowedOrigins(flags.allowedOrigins))` into the conditional streamable block
- Add `app.StreamableModules` to the conditional streamable `di.WithModules(...)` call

**Changes to `src/app/di_test.go`:**
- `TestModule_ProvidesServer_Streamable`: add `StreamableModules` alongside `Module`
- `TestModule_ProvidesServer_Stdio`: remove `fx.Supply(api.RawAllowedOrigins(""))` since `api.Module` is no longer loaded

- [x] modify `src/app/di.go` to extract streamable-only modules into `StreamableModules`
- [x] modify `src/main.go` to include `StreamableModules` and `RawAllowedOrigins` only in the conditional streamable block
- [x] update `src/app/di_test.go` tests for both transport modes
- [x] run project test suite - must pass before next task

### Task 2: Verify acceptance criteria

- [x] run full test suite (`go test ./src/...`)
- [x] run linter (`golangci-lint run ./src/...`)
- [x] verify stdio mode test doesn't load auth modules
- [x] verify streamable mode test still validates full DI graph including auth

### Task 3: Update documentation

- [x] update CLAUDE.md: note that `app.StreamableModules` groups HTTP/auth modules loaded conditionally for streamable transport
- [x] move this plan to `docs/plans/completed/`
