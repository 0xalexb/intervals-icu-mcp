# intervals-icu-mcp

A Model Context Protocol (MCP) server for [Intervals.icu](https://intervals.icu), built in Go. Supports stdio (default) and streamable HTTP transport modes.

## Requirements

- Go 1.25 or later
- golangci-lint (for linting)

## Configuration

Set the following environment variables before running the server:

| Variable | Required | Description |
|----------|----------|-------------|
| `INTERVALS_API_KEY` | Yes | Your Intervals.icu API key (used for Basic auth) |
| `INTERVALS_ATHLETE_ID` | Yes | Your Intervals.icu athlete ID (e.g., `i12345`) |

## Usage

Run the MCP server (stdio transport, default):

```sh
go run ./src/...
```

Run with streamable HTTP transport:

```sh
go run ./src/... --transport streamable --address 127.0.0.1:8080
```

The MCP endpoint is served at `/mcp` (not configurable).

| Flag | Default | Description |
|------|---------|-------------|
| `--transport` | `stdio` | Transport type: `stdio` or `streamable` |
| `--address` | `127.0.0.1:8080` | Listen address for streamable HTTP transport (e.g., `127.0.0.1:8080` or `:9000`) |

Print version:

```sh
go run ./src/... -version
go run ./src/... -v
```

Build with version injection:

```sh
go build -ldflags "-X github.com/0xalexb/hjarta-di.Version=1.0.0 -X github.com/0xalexb/hjarta-di.DIVersion=0.2.1 -X github.com/0xalexb/hjarta-di.CompiledAt=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o intervals-icu-mcp ./src/
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
