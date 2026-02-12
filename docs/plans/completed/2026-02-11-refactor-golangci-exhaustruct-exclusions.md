# Refactor golangci-lint config: exhaustruct exclusions

Refactor .golangci.yml to add exhaustruct exclusion for tool and app paths, removing the ~30 repetitive nolint:exhaustruct annotations.

## Context

- Files involved: `.golangci.yml`, `src/app/tools/*.go`, `src/app/client/client.go`, `src/app/server.go`, `src/app/di.go`
- Related patterns: existing path-based exclusions in `.golangci.yml` for `src/app/tools/`
- Dependencies: none

## Approach

- **Testing approach**: Regular - run linter to verify no regressions
- Complete each task fully before moving to the next
- gochecknoglobals (3 occurrences) and contextcheck (1 occurrence) are left as inline nolint comments since they are isolated cases

## Task 1: Update .golangci.yml with exhaustruct exclusions

**Files:**
- Modify: `.golangci.yml`

- [x] Add `exhaustruct` to the existing `src/app/tools/` path exclusion list
- [x] Add `exhaustruct` exclusion for `src/app/` path (covers server.go, di.go, client/)
- [x] Run linter to verify config is valid: `golangci-lint run ./src/...`

## Task 2: Remove nolint:exhaustruct annotations from source files

**Files:**
- Modify: all `src/app/tools/*.go` files with `nolint:exhaustruct`
- Modify: `src/app/client/client.go`, `src/app/server.go`, `src/app/di.go` if they have `nolint:exhaustruct`

- [x] Remove all `//nolint:exhaustruct` comments from files under `src/app/tools/`
- [x] Remove all `//nolint:exhaustruct` comments from files under `src/app/` (non-test files)
- [x] If a nolint comment covers multiple linters (e.g. `//nolint:exhaustruct,ireturn`), remove only exhaustruct and keep the rest
- [x] Run linter to verify no regressions: `golangci-lint run ./src/...`

## Verification

- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`

## Wrap-up

- [x] Update CLAUDE.md if linting conventions section needs changes
- [x] Move this plan to `docs/plans/completed/`
