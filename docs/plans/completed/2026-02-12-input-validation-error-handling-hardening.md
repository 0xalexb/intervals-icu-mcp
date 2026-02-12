# Input Validation, Error Handling, and Client Hardening

Harden input validation, date format validation, error sanitization, and client edge case coverage across the intervals-icu-mcp server.

## Context

- Files involved:
  - `src/app/tools/tools.go` (add date validation helper, new error sentinels)
  - `src/app/tools/create_event.go` (add missing required field validation)
  - `src/app/tools/get_activities.go` (add date format validation)
  - `src/app/tools/get_events.go` (add date format validation)
  - `src/app/tools/get_wellness.go` (add date format validation)
  - `src/app/tools/get_wellness_trend.go` (add date format validation)
  - `src/app/tools/get_power_curve.go` (add date format validation for optional newest)
  - `src/app/tools/update_event.go` (add date format validation for optional start_date_local)
  - `src/app/client/client.go` (sanitize error messages, handle resp.Body.Close error)
  - `src/app/client/client_test.go` (add edge case tests)
  - All tool test files (add validation test cases)
- Related patterns: existing validation pattern uses sentinel errors from tools.go, returns `nil, nil, err`
- Dependencies: none

## Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Task 1: Add input validation to create_event and date format validation helper

**Files:**
- Modify: `src/app/tools/tools.go`
- Modify: `src/app/tools/create_event.go`

- [x] Add `errMissingName`, `errMissingStartDate`, `errMissingCategory`, `errInvalidDateFormat` sentinel errors to `tools.go`
- [x] Add a `validateDateFormat(s string) error` helper in `tools.go` that checks yyyy-MM-dd format using `time.Parse`
- [x] Add required field validation to `create_event` (name, start_date_local, category must be non-empty)
- [x] Add date format validation for `start_date_local` in `create_event`
- [x] Write tests for the new validation in `create_event_test.go` (missing name, missing start_date_local, missing category, invalid date format)
- [x] Run project test suite - must pass before task 2

## Task 2: Add date format validation across all tools with date parameters

**Files:**
- Modify: `src/app/tools/get_activities.go`
- Modify: `src/app/tools/get_events.go`
- Modify: `src/app/tools/get_wellness.go`
- Modify: `src/app/tools/get_wellness_trend.go`
- Modify: `src/app/tools/get_power_curve.go`
- Modify: `src/app/tools/update_event.go`

- [x] Add date format validation for `oldest` (and `newest` when provided) in `get_activities`
- [x] Add date format validation for `oldest` and `newest` in `get_events`
- [x] Add date format validation for `date` in `get_wellness`
- [x] Add date format validation for `oldest` and `newest` in `get_wellness_trend`
- [x] Add date format validation for `newest` (when provided) in `get_power_curve`
- [x] Add date format validation for `start_date_local` (when provided) in `update_event`
- [x] Add invalid date format test cases to each tool's test file
- [x] Run project test suite - must pass before task 3

## Task 3: Sanitize API error messages and handle resp.Body.Close error

**Files:**
- Modify: `src/app/client/client.go`

- [x] Change error handling on line 125-131: return a generic error message with status code only (e.g. "API request failed: status 404"), do not include the response body
- [x] Replace `_ = resp.Body.Close()` on line 114 with proper error handling using `errors.Join` to combine the close error with any existing error from body reading
- [x] Update existing `TestClient_ErrorResponse` to verify error message does NOT contain API body content
- [x] Add test for `resp.Body.Close` error propagation
- [x] Run project test suite - must pass before task 4

## Task 4: Add client edge case tests

**Files:**
- Modify: `src/app/client/client_test.go`

- [x] Add test for response exceeding `maxResponseSize` (server returns >10MB body)
- [x] Add test for canceled context (`context.WithCancel`, cancel before request)
- [x] Add test for `Get` with nil query params
- [x] Add test for `Delete` with query params
- [x] Run project test suite - must pass

## Verification

- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`

## Wrap-up

- [x] Update CLAUDE.md if internal patterns changed
- [x] Move this plan to `docs/plans/completed/`
