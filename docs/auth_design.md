# OAuth 2.1 Authentication for MCP Server

> **Note**: This is the original design document. The auth package was split on 2026-02-26:
> OAuth HTTP handlers moved to `src/app/api/rest/`, GitHub client moved to `src/app/clients/github/`.
> See `docs/plans/completed/2026-02-26-split-auth-package.md` for details.

## Context

The MCP server currently has no authentication on its `/mcp` endpoint. To serve it on the internet and accept Claude Code clients, we need OAuth 2.1 authentication per the MCP specification. The server will act as both an OAuth Authorization Server (thin proxy to GitHub) and a Resource Server (validating tokens on `/mcp`). GitHub serves as the identity provider, with a username allowlist controlling access. Single-user now (shared Intervals.icu credentials), multi-user ready later.

## Architecture

**OAuth flow (Claude Code perspective):**
1. Claude Code sends request to `/mcp` → gets `401` with `WWW-Authenticate: Bearer resource_metadata=<url>`
2. Fetches `GET /.well-known/oauth-protected-resource` → discovers authorization server
3. Fetches `GET /.well-known/oauth-authorization-server` → discovers endpoints
4. Optionally registers via `POST /oauth/register` (Dynamic Client Registration)
5. Opens browser to `GET /oauth/authorize` with PKCE parameters
6. Server redirects to GitHub OAuth → user authorizes → GitHub redirects to `GET /oauth/callback`
7. Server validates GitHub user against allowlist, generates auth code, redirects to Claude Code's callback
8. Claude Code exchanges code at `POST /oauth/token` → receives JWT + refresh token
9. All subsequent `/mcp` requests include `Authorization: Bearer <jwt>`

**Token strategy:** Server-issued JWTs signed with HMAC-SHA256. Stateless validation via `auth.RequireBearerToken` from go-sdk.

**State:** In-memory for auth codes (10min TTL), refresh tokens (30 days), registered clients. Acceptable for single-instance deployment.

**Auth is mandatory for streamable transport:** The server refuses to start in streamable mode without `--github-client-id`, `--github-client-secret`, and `--auth-issuer` flags. Stdio transport does not use authentication.

## go-sdk types to reuse

- `auth.RequireBearerToken(verifier, opts)` — middleware, protects `/mcp` (`src/auth/auth.go`)
- `auth.TokenVerifier` — callback function type (`src/auth/auth.go`)
- `auth.TokenInfo` — token claims struct with UserID, Scopes, Expiration (`src/auth/auth.go`)
- `auth.TokenInfoFromContext(ctx)` — extract token info in handlers (`src/auth/auth.go`)
- `auth.ProtectedResourceMetadataHandler(metadata)` — serves RFC 9728 at well-known endpoint (`src/auth/auth.go`)
- `oauthex.ProtectedResourceMetadata` — struct (no build tag) (`src/oauthex/oauthex.go`)

**Cannot reuse (behind `//go:build mcp_go_client_oauth`):**
- `oauthex.AuthServerMeta` — define our own `AuthorizationServerMetadata`
- `oauthex.ClientRegistrationMetadata` / `ClientRegistrationResponse` — define our own

## New dependency

- `github.com/golang-jwt/jwt/v5` — JWT signing/verification
- Must be added to depguard allowlist in `.golangci.yml`

## CLI flags to add

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--github-client-id` | string | `""` | GitHub OAuth App client ID |
| `--github-client-secret` | string | `""` | GitHub OAuth App client secret |
| `--allowed-users` | string | `""` | Comma-separated GitHub usernames |
| `--jwt-secret` | string | `""` | HMAC-SHA256 signing secret (auto-generated if empty) |
| `--auth-issuer` | string | `""` | Public URL (e.g., `https://mcp.example.com`) |

Auth is enabled when `--github-client-id` and `--auth-issuer` are both non-empty.

## Files to create

### `src/app/auth/config.go`
Named types for CLI flag values (following `api.RawAllowedOrigins` pattern):
- `GitHubClientID string`
- `GitHubClientSecret string`
- `RawAllowedUsers string` → parsed by `NewAllowedUsers()` → `AllowedUsers []string`
- `JWTSecret string` (auto-generated if empty via `crypto/rand`)
- `Issuer string` — validated by `NewValidatedIssuer()` (must be valid URL with scheme)

`AllowedUsers.Contains(username string) bool` — case-insensitive lookup.

### `src/app/auth/metadata.go`
Our own `AuthorizationServerMetadata` struct (since go-sdk's is behind build tag):
- `issuer`, `authorization_endpoint`, `token_endpoint`, `registration_endpoint`
- `response_types_supported: ["code"]`
- `grant_types_supported: ["authorization_code", "refresh_token"]`
- `token_endpoint_auth_methods_supported: ["none"]`
- `code_challenge_methods_supported: ["S256"]`
- `scopes_supported: ["mcp"]`

`NewAuthorizationServerMetadata(issuer Issuer) *AuthorizationServerMetadata`
`NewProtectedResourceMetadata(issuer Issuer) *oauthex.ProtectedResourceMetadata`

### `src/app/auth/store.go`
In-memory store with `sync.RWMutex`:
- `AuthCode` — code, clientID, redirectURI, codeChallenge, codeChallengeMethod, gitHubUsername, scopes, expiresAt
- `RefreshToken` — token, clientID, gitHubUsername, scopes, expiresAt
- `RegisteredClient` — clientID, redirectURIs, clientName, grantTypes, createdAt
- Methods: `SaveAuthCode`, `ConsumeAuthCode` (one-time use), `SaveRefreshToken`, `ConsumeRefreshToken` (rotation), `SaveClient`, `GetClient`

### `src/app/auth/jwt.go`
JWT issuance and verification using `github.com/golang-jwt/jwt/v5`:
- `IssueAccessToken(secret, issuer, ttl, username, scopes) (string, error)`
- `IssueRefreshToken() (string, error)` — random 32-byte base64url via `crypto/rand`
- `NewTokenVerifier(secret, issuer) auth.TokenVerifier` — returns go-sdk's `TokenVerifier` that maps JWT claims to `auth.TokenInfo` (sets `UserID` = GitHub username for session hijacking prevention in `StreamableHTTPHandler`)

### `src/app/auth/github.go`
GitHub OAuth integration:
- `ExchangeGitHubCode(ctx, clientID, clientSecret, code) (accessToken, error)` — POST to `https://github.com/login/oauth/access_token`
- `GetGitHubUser(ctx, accessToken) (*GitHubUser, error)` — GET `https://api.github.com/user`

### `src/app/auth/handler.go`
HTTP handlers on a `Handler` struct (DI-injected via `HandlerParams` with `fx.In`):

| Endpoint | Method | Handler | Description |
|----------|--------|---------|-------------|
| `/.well-known/oauth-authorization-server` | GET | `HandleAuthServerMetadata` | Serves AS metadata JSON |
| `/oauth/authorize` | GET | `HandleAuthorize` | Validates PKCE params, redirects to GitHub |
| `/oauth/callback` | GET | `HandleCallback` | GitHub callback → validate user → redirect to client |
| `/oauth/token` | POST | `HandleToken` | Exchange auth code or refresh token for JWT |
| `/oauth/register` | POST | `HandleRegister` | Dynamic client registration (RFC 7591) |

State parameter to GitHub: HMAC-signed JSON encoding original authorize request params (clientID, redirectURI, codeChallenge, codeChallengeMethod, originalState, scopes, expiresAt).

PKCE validation: `base64url(sha256(code_verifier)) == stored_code_challenge` using `crypto/subtle.ConstantTimeCompare`.

### `src/app/auth/di.go`
```
var Module = fx.Module("auth", ...)
```
Provides: `AllowedUsers`, validated `Issuer`, `JWTSecret`, `Store`, `Handler`, `AuthorizationServerMetadata`, `*oauthex.ProtectedResourceMetadata`, `auth.TokenVerifier`.

### Test files (one per source file)
- `src/app/auth/config_test.go`
- `src/app/auth/metadata_test.go`
- `src/app/auth/store_test.go`
- `src/app/auth/jwt_test.go`
- `src/app/auth/github_test.go` — with `httptest.Server` mocking GitHub API
- `src/app/auth/handler_test.go` — full OAuth flow integration test

## Files to modify

### `src/main.go`
- Add 5 new CLI flags (github-client-id, github-client-secret, allowed-users, jwt-secret, auth-issuer)
- Supply new auth types into DI via `fx.Supply()`
- If jwt-secret is empty and auth is enabled, generate random 32-byte secret

### `src/app/di.go`
- Add `auth.Module` to the composed module list

### `src/app/api/router.go`
- Change `NewRouter` signature to accept `RouterParams` struct with `fx.In`:
  - `MCPHandler http.Handler` (tag `name:"mcp-raw"`)
  - `Origins AllowedOrigins`
  - `AuthHandler *auth.Handler` (optional — may be nil when auth disabled)
  - `AuthServerMeta *auth.AuthorizationServerMetadata` (optional)
  - `ResourceMeta *oauthex.ProtectedResourceMetadata` (optional)
  - `TokenVerifier auth.TokenVerifier` (optional)
  - `Issuer auth.Issuer` (optional)
- When auth is configured (non-nil handler):
  - Register OAuth/discovery routes
  - Wrap `/mcp` with `auth.RequireBearerToken` middleware (after CORS, before MCP handler)
- When auth is not configured: current behavior (no auth middleware)

### `src/app/api/di.go`
- Remove `fx.ParamTags` from `NewRouter` annotation (struct tags on `RouterParams` handle this)

### `go.mod`
- Add `github.com/golang-jwt/jwt/v5`
- Add `github.com/gofrs/uuid/v5` (for generating client IDs and JTI claims)

### `.golangci.yml`
- Add `github.com/golang-jwt/jwt` and `github.com/gofrs/uuid` to depguard allow lists

## Middleware stack (final, for `/mcp` with auth)

Recovery → RequestID → Logging → RateLimit → MaxRequestSize → CORS → Compress → **RequireBearerToken** → StreamableHTTPHandler

OAuth endpoints (`/oauth/*`, `/.well-known/*`) get layers 1-7 only (no bearer auth).

## Auth activation design

Auth types are supplied to DI only when transport is streamable. The router does not use `optional:"true"` tags since auth is always present in streamable mode and the router only exists in streamable mode. Stdio transport does not activate auth modules.

## Implementation sequence

1. Foundation: `auth/config.go`, `auth/store.go`, `auth/jwt.go` + tests
2. GitHub integration: `auth/github.go` + tests
3. Metadata: `auth/metadata.go` + tests
4. Handlers: `auth/handler.go` + tests
5. DI wiring: `auth/di.go`, update `api/router.go`, `api/di.go`, `app/di.go`, `main.go`
6. Config: `go.mod`, `.golangci.yml`

## Verification

1. **Unit tests:** `go test ./src/app/auth/...`
2. **Build:** `go build ./src/...`
3. **Lint:** `golangci-lint run ./src/...`
4. **Manual test (stdio, no auth):** Run with `--transport stdio` — should work as before
5. **Manual test (streamable, no auth):** Run with `--transport streamable` — should work as before
6. **Manual test (streamable, with auth):**
   - Create a GitHub OAuth App (Settings → Developer settings → OAuth Apps)
   - Run: `./intervals-icu-mcp --transport streamable --github-client-id <id> --github-client-secret <secret> --allowed-users <your-github-username> --auth-issuer https://your-domain.com`
   - Verify `GET /.well-known/oauth-protected-resource` returns metadata JSON
   - Verify `GET /.well-known/oauth-authorization-server` returns metadata JSON
   - Verify `GET /mcp` without token returns 401 with `WWW-Authenticate` header
   - Configure Claude Code: `claude mcp add --transport http intervals-icu https://your-domain.com/mcp`
   - Run `/mcp` in Claude Code to trigger OAuth flow

