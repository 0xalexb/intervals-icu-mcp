# intervals-icu-mcp

A Model Context Protocol (MCP) server for [Intervals.icu](https://intervals.icu), built in Go. Supports stdio (default) and streamable HTTP transport modes.

## Requirements

- Go 1.25 or later
- golangci-lint (for linting)
- A [GitHub OAuth App](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app) (required for streamable transport authentication)

## Configuration

Set the following environment variables before running the server:

| Variable | Required | Description |
|----------|----------|-------------|
| `INTERVALS_API_KEY` | Yes | Your Intervals.icu API key (used for Basic auth) |
| `INTERVALS_ATHLETE_ID` | Yes | Your Intervals.icu athlete ID (e.g., `i12345`) |
| `GITHUB_CLIENT_SECRET` | No | GitHub OAuth app client secret (fallback when `--github-client-secret` flag is empty) |
| `JWT_SECRET` | No | HMAC-SHA256 signing key for JWT tokens (fallback when `--jwt-secret` flag is empty) |

## Usage

Run the MCP server (stdio transport, default):

```sh
go run ./src/...
```

Run with streamable HTTP transport (requires OAuth flags):

```sh
go run ./src/... --transport streamable --address 127.0.0.1:8080 \
  --github-client-id <your-github-client-id> \
  --github-client-secret <your-github-client-secret> \
  --auth-issuer http://localhost:8080 \
  --allowed-users your-github-username
```

The MCP endpoint is served at `/mcp` (not configurable). When using the streamable transport, the endpoint is served behind a middleware stack that provides:

- Panic recovery (returns 500 on unhandled panics)
- Request ID propagation (X-Request-ID header)
- Structured request logging
- Per-IP rate limiting (100 requests/second, burst 200, sliding window)
- Max request body size (1 MB)
- CORS (configurable origins via `--allowed-origins` as full URLs, GET/POST/DELETE/OPTIONS methods)
- Expose-Headers (`Access-Control-Expose-Headers: Mcp-Session-Id` for CORS responses)
- Gzip compression

| Flag | Default | Description |
|------|---------|-------------|
| `--transport` | `stdio` | Transport type: `stdio` or `streamable` |
| `--address` | `127.0.0.1:8080` | Listen address for streamable HTTP transport (e.g., `127.0.0.1:8080` or `:9000`) |
| `--allowed-origins` | _(empty)_ | Comma-separated list of allowed CORS origins as full URLs (e.g., `http://localhost:3000,https://example.com`) |
| `--github-client-id` | _(empty)_ | GitHub OAuth app client ID (required for streamable) |
| `--github-client-secret` | _(empty)_ | GitHub OAuth app client secret (required for streamable; falls back to `GITHUB_CLIENT_SECRET` env var) |
| `--allowed-users` | _(empty)_ | Comma-separated allowed GitHub usernames (empty = allow all authenticated users) |
| `--jwt-secret` | _(empty)_ | HMAC-SHA256 signing key for JWT tokens (auto-generated if empty; falls back to `JWT_SECRET` env var) |
| `--auth-issuer` | _(empty)_ | Issuer URL for the OAuth authorization server (required for streamable). Must be a full URL with http/https scheme |

### Authentication (streamable transport)

OAuth 2.1 authentication is mandatory for the streamable HTTP transport. The server acts as both an OAuth Authorization Server (proxying to GitHub for identity) and a Resource Server (validating JWTs on `/mcp`). Stdio transport does not use authentication.

Create a GitHub OAuth App at GitHub Settings > Developer settings > OAuth Apps, with the callback URL set to `<your-issuer-url>/oauth/callback`.

OAuth endpoints exposed in streamable mode:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/.well-known/oauth-protected-resource` | GET | Protected resource metadata (RFC 9728) |
| `/.well-known/oauth-authorization-server` | GET | Authorization server metadata (RFC 8414) |
| `/oauth/authorize` | GET | Start OAuth authorization flow (redirects to GitHub) |
| `/oauth/callback` | GET | GitHub OAuth callback |
| `/oauth/token` | POST | Token exchange (authorization_code and refresh_token grants) |
| `/oauth/register` | POST | Dynamic client registration (RFC 7591, rate-limited to 2 req/s burst 5) |

Print version:

```sh
go run ./src/... -version
go run ./src/... -v
```

Build with version injection:

```sh
go build -ldflags "-X github.com/0xalexb/hjarta-di.Version=1.0.0 -X github.com/0xalexb/hjarta-di.DIVersion=0.5.0 -X github.com/0xalexb/hjarta-di.CompiledAt=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o intervals-icu-mcp ./src/
```

## MCP Tools

| Tool | Description | Arguments |
|------|-------------|-----------|
| `version` | Returns application build information including version, DI version, and compilation time. | None |
| `get_athlete_profile` | Returns the athlete profile from Intervals.icu. | None |
| `get_activities` | Lists activities in a date range. | `oldest` (required), `newest` (optional) |
| `get_activity_details` | Returns details for a specific activity. | `activity_id` (required) |
| `get_events` | Lists events in a date range. | `oldest` (required), `newest` (required) |
| `create_event` | Creates a new event. | `name`, `start_date_local`, `category` (required); `type`, `description`, `moving_time`, `distance`, `training_load` (optional) |
| `update_event` | Updates an existing event. | `event_id` (required); `name`, `description`, `start_date_local`, `category`, `type`, `moving_time`, `distance`, `training_load` (optional) |
| `delete_event` | Deletes an event by ID. | `event_id` (required) |
| `get_power_curve` | Returns power curve data by activity type. | `type` (required), `newest` (optional) |
| `get_wellness` | Returns wellness data for a specific date. | `date` (required) |
| `get_wellness_trend` | Returns wellness data over a date range. | `oldest` (required), `newest` (required) |

## Development

```sh
go build ./src/...
go test ./src/...
golangci-lint run ./src/...
```
