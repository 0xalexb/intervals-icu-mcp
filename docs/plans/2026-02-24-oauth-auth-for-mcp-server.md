# OAuth 2.1 Authentication for MCP Server

## Overview

Add OAuth 2.1 authentication to the MCP server's streamable HTTP transport. The server acts as both an OAuth Authorization Server (proxying to GitHub for identity) and a Resource Server (validating JWTs on /mcp). Auth is mandatory for streamable HTTP transport - the server refuses to start in streamable mode without auth credentials. Stdio transport is unaffected.

## Context

- Files involved: `src/main.go`, `src/app/di.go`, `src/app/api/router.go`, `src/app/api/di.go`, `src/app/transport_config.go`, `go.mod`, `.golangci.yml`, and new `src/app/auth/` package
- Related patterns: Named types for CLI flags (`api.RawAllowedOrigins`), DI modules (`fx.Module`), validation constructors (`NewAllowedOrigins`)
- Dependencies: `github.com/golang-jwt/jwt/v5` (JWT), `github.com/gofrs/uuid/v5` (client IDs / JTI claims)
- Key go-sdk types to reuse: `auth.RequireBearerToken`, `auth.TokenVerifier`, `auth.TokenInfo`, `auth.ProtectedResourceMetadataHandler`, `oauthex.ProtectedResourceMetadata`
- Types behind build tag (must redefine): `AuthorizationServerMetadata`, `ClientRegistrationMetadata`, `ClientRegistrationResponse`

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- Each new source file gets a corresponding test file
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Auth config types and validation

**Files:**
- Create: `src/app/auth/config.go`
- Create: `src/app/auth/config_test.go`

- [x] Define named types for CLI flag values: `GitHubClientID string`, `GitHubClientSecret string`, `RawAllowedUsers string`, `JWTSecret string`, `Issuer string`
- [x] Implement `NewAllowedUsers(raw RawAllowedUsers) AllowedUsers` - splits by comma, trims whitespace, drops empties, lowercases
- [x] Implement `AllowedUsers.Contains(username string) bool` - case-insensitive lookup
- [x] Implement `NewValidatedIssuer(raw Issuer) (Issuer, error)` - validates URL has scheme (http/https) and non-empty host
- [x] Implement `NewJWTSecret(raw JWTSecret) (JWTSecret, error)` - auto-generates 32-byte random secret via `crypto/rand` if empty
- [x] Write tests for all validation paths (empty input, valid input, invalid URLs, case-insensitive contains)
- [x] Run project test suite - must pass before task 2

### Task 2: In-memory auth store

**Files:**
- Create: `src/app/auth/store.go`
- Create: `src/app/auth/store_test.go`

- [x] Define `AuthCode` struct: code, clientID, redirectURI, codeChallenge, codeChallengeMethod, gitHubUsername, scopes, expiresAt
- [x] Define `RefreshToken` struct: token, clientID, gitHubUsername, scopes, expiresAt
- [x] Define `RegisteredClient` struct: clientID, redirectURIs, clientName, grantTypes, createdAt
- [x] Implement `Store` with `sync.RWMutex` and maps for each entity
- [x] Implement `SaveAuthCode`, `ConsumeAuthCode` (one-time use, checks expiry)
- [x] Implement `SaveRefreshToken`, `ConsumeRefreshToken` (rotation - consume old, caller saves new)
- [x] Implement `SaveClient`, `GetClient`
- [x] Write tests for CRUD operations, one-time consumption, expiry behavior
- [x] Run project test suite - must pass before task 3

### Task 3: JWT issuance and verification

**Files:**
- Create: `src/app/auth/jwt.go`
- Create: `src/app/auth/jwt_test.go`

- [x] Implement `IssueAccessToken(secret JWTSecret, issuer Issuer, ttl time.Duration, username string, scopes []string) (string, error)` using `github.com/golang-jwt/jwt/v5` with HMAC-SHA256
- [x] Include standard claims: iss, sub (username), exp, iat, jti (uuid), scope (space-separated)
- [x] Implement `IssueRefreshToken() (string, error)` - random 32-byte base64url via `crypto/rand`
- [x] Implement `NewTokenVerifier(secret JWTSecret, issuer Issuer) auth.TokenVerifier` - parses JWT, validates signature and claims, maps to `auth.TokenInfo` with `UserID` = GitHub username
- [x] Write tests: issue and verify round-trip, expired token rejection, wrong secret rejection, invalid token format
- [x] Run project test suite - must pass before task 4

### Task 4: GitHub OAuth integration

**Files:**
- Create: `src/app/auth/github.go`
- Create: `src/app/auth/github_test.go`

- [x] Implement `ExchangeGitHubCode(ctx context.Context, clientID GitHubClientID, clientSecret GitHubClientSecret, code string) (string, error)` - POST to `https://github.com/login/oauth/access_token`
- [x] Implement `GetGitHubUser(ctx context.Context, accessToken string) (*GitHubUser, error)` - GET `https://api.github.com/user`, extract login field
- [x] Define `GitHubUser` struct with `Login string`
- [x] Write tests using `httptest.Server` to mock both GitHub endpoints (success and error cases)
- [x] Run project test suite - must pass before task 5

### Task 5: OAuth metadata types

**Files:**
- Create: `src/app/auth/metadata.go`
- Create: `src/app/auth/metadata_test.go`

- [x] Define `AuthorizationServerMetadata` struct (our own, since go-sdk's is behind build tag): issuer, authorization_endpoint, token_endpoint, registration_endpoint, response_types_supported, grant_types_supported, token_endpoint_auth_methods_supported, code_challenge_methods_supported, scopes_supported
- [x] Implement `NewAuthorizationServerMetadata(issuer Issuer) *AuthorizationServerMetadata` - constructs metadata with endpoints derived from issuer
- [x] Implement `NewProtectedResourceMetadata(issuer Issuer) *oauthex.ProtectedResourceMetadata` - sets resource=issuer, authorization_servers=[issuer], scopes_supported=["mcp"], bearer_methods_supported=["header"]
- [x] Write tests verifying correct endpoint construction and JSON serialization
- [x] Run project test suite - must pass before task 6

### Task 6: OAuth HTTP handlers

**Files:**
- Create: `src/app/auth/handler.go`
- Create: `src/app/auth/handler_test.go`

- [ ] Define `Handler` struct with DI-injected dependencies (`HandlerParams` with `fx.In`): Store, AllowedUsers, GitHubClientID, GitHubClientSecret, JWTSecret, Issuer, AuthorizationServerMetadata
- [ ] Implement `HandleAuthServerMetadata` (GET `/.well-known/oauth-authorization-server`) - serves AS metadata JSON
- [ ] Implement `HandleAuthorize` (GET `/oauth/authorize`) - validates PKCE params (response_type=code, code_challenge, code_challenge_method=S256, client_id, redirect_uri, state, scope), generates HMAC-signed state containing original params, redirects to GitHub OAuth authorize URL
- [ ] Implement `HandleCallback` (GET `/oauth/callback`) - validates HMAC state, exchanges GitHub code for access token, fetches GitHub user, checks allowlist, generates auth code, stores it, redirects to client's redirect_uri with code and state
- [ ] Implement `HandleToken` (POST `/oauth/token`) - handles grant_type=authorization_code (validates PKCE code_verifier via constant-time compare, issues JWT + refresh token) and grant_type=refresh_token (rotates refresh token, issues new JWT)
- [ ] Implement `HandleRegister` (POST `/oauth/register`) - dynamic client registration per RFC 7591, generates client_id via uuid, stores client
- [ ] Write integration tests covering the full OAuth flow: register -> authorize -> callback -> token exchange -> refresh
- [ ] Run project test suite - must pass before task 7

### Task 7: Auth DI module and wiring

**Files:**
- Create: `src/app/auth/di.go`
- Modify: `src/app/di.go`
- Modify: `src/app/api/router.go`
- Modify: `src/app/api/di.go`
- Modify: `src/main.go`
- Modify: `go.mod`
- Modify: `.golangci.yml`

- [ ] Create `auth.Module` (`fx.Module("auth", ...)`) providing: AllowedUsers, validated Issuer, JWTSecret, Store, Handler, AuthorizationServerMetadata, *oauthex.ProtectedResourceMetadata, auth.TokenVerifier
- [ ] Add startup validation in `src/main.go`: when `--transport streamable`, require `--github-client-id` and `--auth-issuer` to be non-empty; exit with a clear error message if missing (e.g., "streamable transport requires --github-client-id and --auth-issuer flags")
- [ ] When `--transport stdio`, ignore auth flags entirely (auth does not apply to stdio)
- [ ] Add 5 CLI flags to `src/main.go`: `--github-client-id`, `--github-client-secret`, `--allowed-users`, `--jwt-secret`, `--auth-issuer`; supply new named types via `fx.Supply()`
- [ ] Only supply auth types and include `auth.Module` when transport is streamable (conditional DI based on transport, not on whether flags happen to be set)
- [ ] Add `auth.Module` to `app.Module` composition in `src/app/di.go` (always included; when stdio, the auth types simply won't be supplied so the module provides nothing)
- [ ] Change `NewRouter` in `src/app/api/router.go` to accept `RouterParams` struct with `fx.In`: MCPHandler (tag `name:"mcp-raw"`), Origins, auth.Handler, auth.AuthorizationServerMetadata, *oauthex.ProtectedResourceMetadata, auth.TokenVerifier, auth.Issuer - no `optional:"true"` tags since auth is always present in streamable mode and router only exists in streamable mode
- [ ] Register `/.well-known/oauth-protected-resource` (using `auth.ProtectedResourceMetadataHandler`), `/.well-known/oauth-authorization-server`, `/oauth/authorize`, `/oauth/callback`, `/oauth/token`, `/oauth/register`; wrap `/mcp` with `auth.RequireBearerToken` middleware (ResourceMetadataURL pointing to `/.well-known/oauth-protected-resource`)
- [ ] Update `src/app/api/di.go`: remove `fx.ParamTags` from NewRouter annotation (struct tags on RouterParams handle injection)
- [ ] Add `github.com/golang-jwt/jwt/v5` and `github.com/gofrs/uuid/v5` to `go.mod` via `go get`
- [ ] Add `github.com/golang-jwt/jwt` and `github.com/gofrs/uuid` to depguard allow lists in `.golangci.yml` (both main and tests rules)
- [ ] Add `src/app/auth/` path exclusion for `exhaustruct` in `.golangci.yml` (consistent with `src/app/` exclusion)
- [ ] Write tests verifying: streamable mode fails to start without auth flags, streamable mode starts with auth flags, stdio mode starts without auth flags
- [ ] Run project test suite - must pass before task 8

### Task 8: Verify acceptance criteria

- [ ] Manual test: run with `--transport stdio` (no auth flags) - verify stdio works as before
- [ ] Manual test: run with `--transport streamable` without auth flags - verify server refuses to start with clear error
- [ ] Manual test: run with `--transport streamable` with all auth flags - verify server starts and serves OAuth endpoints
- [ ] Run full test suite: `go test ./src/...`
- [ ] Run linter: `golangci-lint run ./src/...`
- [ ] Verify test coverage meets 80%+

### Task 9: Update documentation

- [ ] Update CLAUDE.md with auth package architecture, new CLI flags, and mandatory auth for streamable transport
- [ ] Move this plan to `docs/plans/completed/`
