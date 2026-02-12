# Refactor: Add DI module to tools package per golang conventions

The golang rules require that each folder has its own `di.go` file and that a DI module groups all DI modules for its immediate subfolders. Currently, `src/app/tools/` has no `di.go` file — tool constructors are wired directly from the parent `src/app/di.go`. This plan extracts a `tools.Module` into its own `di.go` and updates the parent module to compose it.

- Files involved: `src/app/tools/di.go` (new), `src/app/di.go`, `src/app/di_test.go`, `CLAUDE.md`
- Related patterns: `fx.Module` as package variable, fx group tags for tool collection
- Dependencies: none

## Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- The `tools.Module` will own all fx.Provide calls for tool constructors with group tags
- The parent `app.Module` will include `tools.Module` instead of directly referencing individual tool constructors
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Task 1: Create `tools.Module` in `src/app/tools/di.go`

**Files:**
- Create: `src/app/tools/di.go`

### Steps

- [x] Create `src/app/tools/di.go` with a package-level `Module` variable (`fx.Module("tools", ...)`)
- [x] Move the `fx.Provide(fx.Annotate(tools.NewVersionTool, fx.ResultTags(...)))` line from `src/app/di.go` into the new `tools.Module`
- [x] Create `src/app/tools/di_test.go` with a test that verifies `tools.Module` provides at least one `ToolRegistration` into the fx group
- [x] Run `go test ./src/...` — must pass before task 2

## Task 2: Update `app.Module` to compose `tools.Module`

**Files:**
- Modify: `src/app/di.go`
- Modify: `src/app/di_test.go`

### Steps

- [x] In `src/app/di.go`, remove the direct `fx.Provide(fx.Annotate(tools.NewVersionTool, ...))` line
- [x] Add `tools.Module` as a composed sub-module inside `app.Module` (using `fx.Module` nesting or `fx.Options`)
- [x] Verify `src/app/di_test.go` still passes — the existing `TestModule_ProvidesServer` test should continue to work since the tools are still provided via the group
- [x] Run `go test ./src/...` — must pass before task 3

## Task 3: Update documentation

**Files:**
- Modify: `CLAUDE.md`

### Steps

- [x] Update the CLAUDE.md Architecture section: remove the statement that `src/app/tools/` does not have its own `di.go`
- [x] Document that `src/app/tools/di.go` contains `tools.Module` which provides tool constructors via fx group, and `app.Module` composes `tools.Module`

## Verification

- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`
- [x] Move this plan to `docs/plans/completed/`
