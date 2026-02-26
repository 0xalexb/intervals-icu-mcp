# Split Auth Package: Handlers to api/rest, GitHub Client to clients/github

## Overview

Refactor the `src/app/auth/` package by extracting OAuth HTTP handlers into `src/app/api/rest/` and the GitHub OAuth client into `src/app/clients/github/`. Store, config, JWT, and metadata logic remain in `src/app/auth/`.

## Context

- Files involved:
  - `src/app/auth/handler.go` -> moves to `src/app/api/rest/handler.go`
  - `src/app/auth/handler_test.go` -> moves to `src/app/api/rest/handler_test.go`
  - `src/app/auth/github.go` -> moves to `src/app/clients/github/github.go`
  - `src/app/auth/github_test.go` -> moves to `src/app/clients/github/github_test.go`
  - `src/app/auth/di.go` -> updated (remove Handler and GitHubClient provisions)
  - `src/app/auth/config.go` -> updated (add scheme constants that handler.go currently defines)
  - `src/app/api/router.go` -> updated (import Handler from rest instead of auth)
  - `src/app/api/router_test.go` -> updated (import changes)
  - `src/app/di.go` -> updated (compose new modules)
  - `.golangci.yml` -> updated (add exclusions for new paths)
- Related patterns: flat DI module composition at `app.Module` level, `testing.go` pattern for test clients
- Dependencies: no new external dependencies

## Development Approach

- **Testing approach**: Regular (code first, then tests) - tests are being moved, not written from scratch
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Extract GitHub client to `src/app/clients/github/`

**Files:**
- Create: `src/app/clients/github/github.go`
- Create: `src/app/clients/github/testing.go`
- Create: `src/app/clients/github/di.go`
- Create: `src/app/clients/github/github_test.go`

- [x] Create `src/app/clients/github/github.go` with `GitHubClient`, `GitHubUser` types and `ExchangeGitHubCode`, `GetGitHubUser` methods. Change method parameters from `auth.GitHubClientID`/`auth.GitHubClientSecret` named types to plain `string` parameters to avoid coupling the GitHub client to auth config types.
- [x] Create `src/app/clients/github/testing.go` with `NewTestClient(tokenURL, userURL string) *GitHubClient` helper (follows existing `client/testing.go` pattern).
- [x] Create `src/app/clients/github/di.go` with `var Module = fx.Module("github", fx.Provide(NewGitHubClient))`.
- [x] Create `src/app/clients/github/github_test.go` - adapt tests from `auth/github_test.go`, updating type references to use the new package's types.
- [x] Run `go test ./src/app/clients/github/...` - must pass before task 2

### Task 2: Extract OAuth handlers to `src/app/api/rest/`

**Files:**
- Create: `src/app/api/rest/handler.go`
- Create: `src/app/api/rest/di.go`
- Create: `src/app/api/rest/handler_test.go`

- [ ] Create `src/app/api/rest/handler.go` with `Handler`, `HandlerParams`, and all Handle* methods plus private helpers. `HandlerParams` references types from `auth` (`*auth.Store`, `auth.AllowedUsers`, `auth.GitHubClientID`, `auth.GitHubClientSecret`, `auth.JWTSecret`, `auth.Issuer`, `*auth.AuthorizationServerMetadata`) and `clients/github` (`*github.GitHubClient`). Where handler calls `GitHubClient.ExchangeGitHubCode`, cast named types to `string`: `string(h.ghClientID)`. Where handler calls `auth.IssueAccessToken`, `auth.IssueRefreshToken`, `auth.NewAuthorizationServerMetadata` - use qualified imports.
- [ ] Create `src/app/api/rest/di.go` with `var Module = fx.Module("rest", fx.Provide(NewHandler))`.
- [ ] Create `src/app/api/rest/handler_test.go` - adapt tests from `auth/handler_test.go`, updating type references to use `auth.*` and `github.NewTestClient()` for the GitHub client. Use `auth.NewStore()`, `auth.AllowedUsers{}`, etc.
- [ ] Run `go test ./src/app/api/rest/...` - must pass before task 3

### Task 3: Rewire DI, update imports, and remove old files

**Files:**
- Modify: `src/app/auth/config.go`
- Modify: `src/app/auth/di.go`
- Modify: `src/app/api/router.go`
- Modify: `src/app/api/router_test.go`
- Modify: `src/app/di.go`
- Delete: `src/app/auth/handler.go`
- Delete: `src/app/auth/handler_test.go`
- Delete: `src/app/auth/github.go`
- Delete: `src/app/auth/github_test.go`

- [ ] Add `schemeHTTP` and `schemeHTTPS` constants to `src/app/auth/config.go` (currently defined in `handler.go` and used by `config.go`'s `NewValidatedIssuer`)
- [ ] Update `src/app/auth/di.go`: remove `fx.Provide(NewHandler)` and `fx.Provide(NewGitHubClient)` from the Module
- [ ] Update `src/app/api/router.go`: change import from `appauth.Handler` to `rest.Handler` (import `"github.com/0xalexb/intervals-icu-mcp/src/app/api/rest"`), update `RouterParams.AuthHandler` type to `*rest.Handler`
- [ ] Update `src/app/api/router_test.go`: change handler construction to use `rest.NewHandler(rest.HandlerParams{...})`, import `rest` package and `clients/github` package
- [ ] Update `src/app/di.go`: add imports for `ghclient "github.com/0xalexb/intervals-icu-mcp/src/app/clients/github"` and `"github.com/0xalexb/intervals-icu-mcp/src/app/api/rest"`, add `ghclient.Module` and `rest.Module` to the Module composition
- [ ] Delete `src/app/auth/handler.go`, `src/app/auth/handler_test.go`, `src/app/auth/github.go`, `src/app/auth/github_test.go`
- [ ] Run `go test ./src/...` - must pass before task 4

### Task 4: Update linter config

**Files:**
- Modify: `.golangci.yml`

- [ ] Add linter exclusion for `src/app/api/rest/` matching `src/app/auth/` exclusions: `exhaustruct`, `tagliatelle`, `gosec`, `varnamelen`, `noinlineerr`, `funcorder`
- [ ] Add linter exclusion for `src/app/clients/github/`: `exhaustruct`, `tagliatelle`, `gosec`, `varnamelen`, `noinlineerr`
- [ ] Add text exclusion "avoid meaningless package names" for `src/app/clients/` path (package name `github` is intentional)
- [ ] Run `go test ./src/...` - confirm tests still pass

### Task 5: Verify acceptance criteria

- [ ] Run full test suite: `go test ./src/...`
- [ ] Run linter: `golangci-lint run ./src/...`
- [ ] Verify no circular dependencies between packages
- [ ] Verify `src/app/auth/` contains only: `config.go`, `config_test.go`, `store.go`, `store_test.go`, `jwt.go`, `jwt_test.go`, `metadata.go`, `metadata_test.go`, `di.go`
- [ ] Verify `src/app/clients/github/` contains: `github.go`, `github_test.go`, `testing.go`, `di.go`
- [ ] Verify `src/app/api/rest/` contains: `handler.go`, `handler_test.go`, `di.go`

### Task 6: Update documentation

- [ ] Update CLAUDE.md: document new package structure (`src/app/clients/github/`, `src/app/api/rest/`), update auth package description, update DI module composition description, add linter exclusion notes for new paths
- [ ] Move this plan to `docs/plans/completed/`
