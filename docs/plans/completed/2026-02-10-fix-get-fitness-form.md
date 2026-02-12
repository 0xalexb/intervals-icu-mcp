# Fix get_fitness_form Tool - Correct API Endpoint and Field Names

Fix get_fitness_form tool to use the correct Intervals.icu API endpoint and field names. The tool currently calls `/api/v1/athlete/{id}` (athlete profile) which doesn't contain fitness data. It should call `/api/v1/athlete/{id}/wellness/{date}` for today's date, and use the correct JSON field names (`ctLoad`, `atlLoad`) from the wellness response.

## Context

- Files involved:
  - `src/app/tools/get_fitness_form.go`
  - `src/app/tools/get_fitness_form_test.go`
- Related patterns: `get_wellness.go` uses the same `/wellness/{date}` endpoint pattern
- Dependencies: none

## Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Task 1: Fix the fitness form tool implementation

**Files:**
- Modify: `src/app/tools/get_fitness_form.go`
- Modify: `src/app/tools/get_fitness_form_test.go`

Changes to `get_fitness_form.go`:
- Change the API endpoint from `/api/v1/athlete/{id}` to `/api/v1/athlete/{id}/wellness/{today}` where today is `time.Now().Format("2006-01-02")`
- Update `fitnessFormResponse` JSON tags: `ctl` -> `ctLoad`, `atl` -> `atlLoad`, `rampRate` -> verify correct field name
- Update error message from "fetching athlete profile for fitness form" to "fetching wellness for fitness form"

Changes to `get_fitness_form_test.go`:
- Update mock server to expect path `/api/v1/athlete/{id}/wellness/{date}` instead of `/api/v1/athlete/{id}`
- Update responseJSON to use correct field names: `ctLoad`, `atlLoad`
- Verify `rampRate` field name is correct in the wellness response

- [x] Update `fitnessFormResponse` struct with correct JSON field names from wellness API
- [x] Change API endpoint to `/api/v1/athlete/{id}/wellness/{today}`
- [x] Update test mock to match new endpoint and response format
- [x] Run `go test ./src/app/tools/...` - must pass

## Task 2: Verification

- [x] Run `go test ./src/...`
- [x] Run `golangci-lint run ./src/...`

## Wrap-up

- [x] Move this plan to `docs/plans/completed/`
