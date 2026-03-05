  Consolidated Security Audit Report

  Four parallel agents investigated OAuth flow, JWT/tokens, store/sessions, and HTTP transport security. Here are the deduplicated findings, ranked by severity.

  ---
  HIGH

  1. Login CSRF — Attacker Can Obtain Token Representing a Victim

  Files: handler.go:HandleAuthorize, handler.go:HandleCallback

  An attacker registers their own client, crafts an authorize URL with their PKCE pair, and tricks a victim into clicking it. The victim authenticates with GitHub, and the auth code is redirected to the attacker's redirect_uri.
  The attacker exchanges it with their known code_verifier and gets a JWT with sub=victim. PKCE prevents code interception, not login CSRF — the attacker controls both sides of the PKCE pair.

  Mitigation: Bind the authorization request to a browser session (server-side nonce in a signed cookie or session store), or add a server-rendered consent page before the GitHub redirect.

  1. CORS Hostname Matching Strips Port

  Files: origins.go:57-69, router.go:49

  Hostnames() strips the port, so allowing <http://localhost:3000> also allows <http://localhost:9999>. Any process on the same host can make cross-origin requests to /mcp.

  Mitigation: Match full origin (scheme + host + port) instead of hostname only.

  1. Credentials Visible in Process Listing

  File: main.go:51-59

  --github-client-secret and --jwt-secret are CLI flags visible via ps aux / /proc/cmdline. A leaked JWT secret allows forging tokens.

  Mitigation: You mentioned planning SQLite/Vault integration — that would resolve this. Alternatively, read from environment variables or a config file.

  ---
  MEDIUM

  1. Unauthenticated Registration DoS

  Files: handler.go:HandleRegister, store.go:maxClients=1000

  POST /oauth/register is unauthenticated. An attacker fills 1000 client slots in ~10 seconds. Clients persist 30 days. Legitimate registrations blocked with 503.

  Mitigation: Per-endpoint rate limiting on /oauth/register, optional --registration-token flag, or shorter client TTL.

  1. Auth Code Consumed Before PKCE/Client Validation

  File: handler.go:507-518

  ConsumeAuthCode deletes the code before verifying code_verifier, client_id, and redirect_uri. An attacker who observes the code can burn it with a wrong verifier, forcing the legitimate user to restart auth.

  Mitigation: Validate PKCE and binding checks before consuming the code, or use a read-then-validate-then-delete pattern.

  1. No Cap on Auth Codes / Refresh Tokens in Store

  File: store.go:SaveAuthCode, store.go:SaveRefreshToken

  Unlike clients (capped at 1000), auth codes and refresh tokens have no size limit. Between 5-minute cleanup cycles, tokens can accumulate. Practical impact is limited by GitHub's rate limits on the OAuth flow.

  Mitigation: Add caps similar to maxClients, or add per-user limits.

  1. Global Rate Limiting Is Not Per-IP

  File: router.go:46

  Single global token bucket — one attacker consuming 100 req/s starves all other users.

  Mitigation: Per-IP rate limiting.

  1. JWT Secret Dual-Use for State HMAC

  Files: handler.go:756, jwt.go:50

  Same key used for JWT signing and OAuth state HMAC. Key compromise affects both.

  Mitigation: Derive a separate state key via HKDF, or add a dedicated --state-secret flag.

  1. GitHub Error Parameters Reflected Verbatim

  File: handler.go:364-368

  GitHub's error/error_description forwarded directly into JSON responses. Content injection risk for OAuth clients parsing errors.

  Mitigation: Allowlist known error codes; sanitize descriptions.

  ---
  LOW

  ┌─────┬──────────────────────────────────────────────────────────────┬───────────────────────────┐
  │  #  │                           Finding                            │           File            │
  ├─────┼──────────────────────────────────────────────────────────────┼───────────────────────────┤
  │ L1  │ Unnecessary DELETE method in CORS config                     │ router.go:50              │
  ├─────┼──────────────────────────────────────────────────────────────┼───────────────────────────┤
  │ L2  │ localhost accepted as loopback (may resolve to non-loopback) │ handler.go:isLoopbackHost │
  ├─────┼──────────────────────────────────────────────────────────────┼───────────────────────────┤
  │ L3  │ State replayable within 10-min TTL (no nonce revocation)     │ handler.go:verifyState    │
  ├─────┼──────────────────────────────────────────────────────────────┼───────────────────────────┤
  │ L4  │ Missing nbf claim in JWT                                     │ jwt.go:39-46              │
  ├─────┼──────────────────────────────────────────────────────────────┼───────────────────────────┤
  │ L5  │ Expired token error leaks existence info                     │ store.go:183-186          │
  ├─────┼──────────────────────────────────────────────────────────────┼───────────────────────────┤
  │ L6  │ NewTestClient exported without timeout                       │ clients/github/testing.go │
  ├─────┼──────────────────────────────────────────────────────────────┼───────────────────────────┤
  │ L7  │ No JTI tracking (no token revocation)                        │ jwt.go                    │
  └─────┴──────────────────────────────────────────────────────────────┴───────────────────────────┘

  ---
  Confirmed Secure

  The audit confirmed these are correctly implemented:

- PKCE S256 with subtle.ConstantTimeCompare
- JWT algorithm pinning (HS256 only, alg:none blocked)
- JWT claims validation (iss, sub, exp all required)
- Auth code one-time use with mutex protection
- Redirect URI exact-match + fragment rejection + HTTP loopback-only
- Refresh token rotation with client_id binding
- State HMAC with constant-time comparison + 10-min TTL
- Scope allowlist (only "mcp" accepted end-to-end)
- Bearer middleware on both /mcp and /mcp/ (no bypass)
- Cache-Control: no-store on token responses
- GitHub access token never persisted or logged
- JWT secret entropy (256-bit via crypto/rand)
- Concurrency/locking strategy (correct, no deadlocks)

  ---
  The top priorities to address are H1 (Login CSRF), H2 (CORS port bypass), and H3 (credential exposure). Your planned SQLite/Vault work will directly address H3. Want me to create an implementation plan for any of these fixes?
