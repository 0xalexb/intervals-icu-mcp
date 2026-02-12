# Move TransportConfig DI provider out of main.go

## Overview

Move the inline fx.Provide for TransportConfig from main.go into the app package, following the DI rule: "Avoid creating DI modules in main.go."

## Context

- Files involved: `src/main.go`, `src/app/di.go`, `src/app/transport_config.go`
- Related patterns: existing DI module pattern in `src/app/di.go`, `src/app/client/di.go`, `src/app/tools/di.go`
- Dependencies: none

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Move TransportConfig provider to app package and simplify main.go

**Files:**
- Modify: `src/app/transport_config.go`
- Modify: `src/app/di.go`
- Modify: `src/main.go`

- [x] Add a constructor function `NewTransportConfig(transport, basePath string) TransportConfig` in `src/app/transport_config.go`
- [x] Add `fx.Provide` for TransportConfig in `app.Module` in `src/app/di.go`, using fx.Supply or accepting it as an fx.Provide from main via `di.WithModules(fx.Supply(cfg.transportConfig))`
- [x] In `src/main.go`, replace `fx.Provide(func() app.TransportConfig { return cfg.transportConfig })` with `fx.Supply(cfg.transportConfig)` - this uses fx.Supply which is declarative value injection, not module creation
- [x] Verify existing tests still pass
- [x] Write a unit test for the TransportConfig constructor if one is added

### Task 2: Verify acceptance criteria

- [x] run full test suite: `go test ./src/...`
- [x] run linter: `golangci-lint run ./src/...`
- [x] verify the inline fx.Provide is removed from main.go

### Task 3: Update documentation

- [x] update CLAUDE.md if internal patterns changed
- [x] move this plan to `docs/plans/completed/`
