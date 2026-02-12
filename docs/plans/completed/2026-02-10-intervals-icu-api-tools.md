# Build Intervals.icu API Tools

Build 11 Intervals.icu API tools: get_athlete_profile, get_fitness_form, get_activities, get_activity_details, get_events, create_event, update_event, delete_event, get_power_curve, get_wellness, get_wellness_trend. Add an HTTP client layer that authenticates via Basic auth using environment variables INTERVALS_API_KEY and INTERVALS_ATHLETE_ID.

- Files involved:
  - Create: `src/app/client/client.go` — HTTP client for Intervals.icu API
  - Create: `src/app/client/di.go` — DI module for client
  - Create: `src/app/client/client_test.go` — Client tests
  - Create: `src/app/tools/get_athlete_profile.go`
  - Create: `src/app/tools/get_fitness_form.go`
  - Create: `src/app/tools/get_activities.go`
  - Create: `src/app/tools/get_activity_details.go`
  - Create: `src/app/tools/get_events.go`
  - Create: `src/app/tools/create_event.go`
  - Create: `src/app/tools/update_event.go`
  - Create: `src/app/tools/delete_event.go`
  - Create: `src/app/tools/get_power_curve.go`
  - Create: `src/app/tools/get_wellness.go`
  - Create: `src/app/tools/get_wellness_trend.go`
  - Create: test files for each tool (one per tool file)
  - Modify: `src/app/tools/di.go` — register all new tools
  - Modify: `src/app/di.go` — compose client.Module
  - Modify: `.golangci.yml` — add depguard allow if needed
- Related patterns: ToolRegistration function pattern from version.go, fx group "mcp_tools", fx.Module composition
- Dependencies: No new external dependencies (use net/http from stdlib)

## Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- Each tool constructor receives the client via DI (fx injects it)
- Tools return JSON responses as TextContent
- The HTTP client handles Basic auth, base URL construction, and JSON marshaling
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Task 1: Create the Intervals.icu HTTP client

Create `src/app/client/` package with a Client struct that wraps net/http.Client and handles authentication, base URL, and JSON request/response.

**Files:**
- Create: `src/app/client/client.go`
- Create: `src/app/client/di.go`
- Create: `src/app/client/client_test.go`
- Modify: `src/app/di.go` — add client.Module
- Modify: `.golangci.yml` — add depguard allow for encoding/json if needed (it's in $gostd so should be fine)

### Steps

- [x] Define Config struct with APIKey and AthleteID fields, read from env vars INTERVALS_API_KEY and INTERVALS_ATHLETE_ID
- [x] Define Client struct with baseURL, apiKey, athleteID, and *http.Client fields
- [x] Implement NewClient(cfg Config) constructor that validates config and builds the client
- [x] Implement Client.Get(ctx, path, queryParams) method that sends GET requests with Basic auth and returns the response body as []byte
- [x] Implement Client.Post(ctx, path, body) method for POST requests with JSON body
- [x] Implement Client.Put(ctx, path, body) method for PUT requests with JSON body
- [x] Implement Client.Delete(ctx, path, queryParams) method for DELETE requests
- [x] Create client.Module in di.go that provides Config (from env) and Client
- [x] Compose client.Module into app.Module in src/app/di.go
- [x] Write tests: NewClient validation, request construction (method, URL, auth header, body), using httptest.Server for round-trip tests
- [x] Run `go test ./src/...` — must pass before task 2

## Task 2: Implement get_athlete_profile tool

**Files:**
- Create: `src/app/tools/get_athlete_profile.go`
- Create: `src/app/tools/get_athlete_profile_test.go`
- Modify: `src/app/tools/di.go` — register tool

### Steps

- [x] Define getAthleteProfileArgs struct (empty — uses athlete ID from config)
- [x] Implement NewGetAthleteProfileTool(client) ToolRegistration that calls GET /api/v1/athlete/{id}
- [x] Return response JSON as TextContent
- [x] Register in tools.Module via fx group
- [x] Write tests using httptest.Server to mock the API response
- [x] Run `go test ./src/...` — must pass before task 3

## Task 3: Implement get_fitness_form tool

**Files:**
- Create: `src/app/tools/get_fitness_form.go`
- Create: `src/app/tools/get_fitness_form_test.go`
- Modify: `src/app/tools/di.go` — register tool

### Steps

- [x] Define getFitnessFormArgs struct (empty — returns current ATL/CTL/TSB from athlete profile)
- [x] Implement NewGetFitnessFormTool(client) ToolRegistration that calls GET /api/v1/athlete/{id} and extracts ctl, atl, ramp_rate fields, computes tsb = ctl - atl
- [x] Return formatted fitness/form data as TextContent
- [x] Register in tools.Module
- [x] Write tests
- [x] Run `go test ./src/...` — must pass before task 4

## Task 4: Implement get_activities tool

**Files:**
- Create: `src/app/tools/get_activities.go`
- Create: `src/app/tools/get_activities_test.go`
- Modify: `src/app/tools/di.go` — register tool

### Steps

- [x] Define getActivitiesArgs struct with Oldest (required string), Newest (optional string) fields using jsonschema tags
- [x] Implement NewGetActivitiesTool(client) ToolRegistration that calls GET /api/v1/athlete/{id}/activities with oldest/newest query params
- [x] Return response JSON as TextContent
- [x] Register in tools.Module
- [x] Write tests
- [x] Run `go test ./src/...` — must pass before task 5

## Task 5: Implement get_activity_details tool

**Files:**
- Create: `src/app/tools/get_activity_details.go`
- Create: `src/app/tools/get_activity_details_test.go`
- Modify: `src/app/tools/di.go` — register tool

### Steps

- [x] Define getActivityDetailsArgs struct with ActivityID (required string) field
- [x] Implement NewGetActivityDetailsTool(client) ToolRegistration that calls GET /api/v1/activity/{activityId}
- [x] Return response JSON as TextContent
- [x] Register in tools.Module
- [x] Write tests
- [x] Run `go test ./src/...` — must pass before task 6

## Task 6: Implement get_events tool

**Files:**
- Create: `src/app/tools/get_events.go`
- Create: `src/app/tools/get_events_test.go`
- Modify: `src/app/tools/di.go` — register tool

### Steps

- [x] Define getEventsArgs struct with Oldest (required), Newest (required) fields
- [x] Implement NewGetEventsTool(client) ToolRegistration that calls GET /api/v1/athlete/{id}/events with oldest/newest query params
- [x] Return response JSON as TextContent
- [x] Register in tools.Module
- [x] Write tests
- [x] Run `go test ./src/...` — must pass before task 7

## Task 7: Implement create_event tool

**Files:**
- Create: `src/app/tools/create_event.go`
- Create: `src/app/tools/create_event_test.go`
- Modify: `src/app/tools/di.go` — register tool

### Steps

- [x] Define createEventArgs struct with Name (required), StartDateLocal (required), Category (required), Type, Description, MovingTime, Distance, TrainingLoad (optional) fields
- [x] Implement NewCreateEventTool(client) ToolRegistration that calls POST /api/v1/athlete/{id}/events with JSON body
- [x] Return response JSON as TextContent
- [x] Register in tools.Module
- [x] Write tests verifying request body construction and response handling
- [x] Run `go test ./src/...` — must pass before task 8

## Task 8: Implement update_event tool

**Files:**
- Create: `src/app/tools/update_event.go`
- Create: `src/app/tools/update_event_test.go`
- Modify: `src/app/tools/di.go` — register tool

### Steps

- [x] Define updateEventArgs struct with EventID (required), Name, Description, StartDateLocal, Category, Type, MovingTime, Distance, TrainingLoad (all optional) fields
- [x] Implement NewUpdateEventTool(client) ToolRegistration that calls PUT /api/v1/athlete/{id}/events/{eventId} with JSON body
- [x] Return response JSON as TextContent
- [x] Register in tools.Module
- [x] Write tests
- [x] Run `go test ./src/...` — must pass before task 9

## Task 9: Implement delete_event tool

**Files:**
- Create: `src/app/tools/delete_event.go`
- Create: `src/app/tools/delete_event_test.go`
- Modify: `src/app/tools/di.go` — register tool

### Steps

- [x] Define deleteEventArgs struct with EventID (required) field
- [x] Implement NewDeleteEventTool(client) ToolRegistration that calls DELETE /api/v1/athlete/{id}/events/{eventId}
- [x] Return confirmation TextContent
- [x] Register in tools.Module
- [x] Write tests
- [x] Run `go test ./src/...` — must pass before task 10

## Task 10: Implement get_power_curve tool

**Files:**
- Create: `src/app/tools/get_power_curve.go`
- Create: `src/app/tools/get_power_curve_test.go`
- Modify: `src/app/tools/di.go` — register tool

### Steps

- [x] Define getPowerCurveArgs struct with Type (required, e.g. "Ride"), Newest (optional) fields
- [x] Implement NewGetPowerCurveTool(client) ToolRegistration that calls GET /api/v1/athlete/{id}/power-curves.json with type query param
- [x] Return response JSON as TextContent
- [x] Register in tools.Module
- [x] Write tests
- [x] Run `go test ./src/...` — must pass before task 11

## Task 11: Implement get_wellness tool

**Files:**
- Create: `src/app/tools/get_wellness.go`
- Create: `src/app/tools/get_wellness_test.go`
- Modify: `src/app/tools/di.go` — register tool

### Steps

- [x] Define getWellnessArgs struct with Date (required, yyyy-MM-dd) field
- [x] Implement NewGetWellnessTool(client) ToolRegistration that calls GET /api/v1/athlete/{id}/wellness/{date}
- [x] Return response JSON as TextContent
- [x] Register in tools.Module
- [x] Write tests
- [x] Run `go test ./src/...` — must pass before task 12

## Task 12: Implement get_wellness_trend tool

**Files:**
- Create: `src/app/tools/get_wellness_trend.go`
- Create: `src/app/tools/get_wellness_trend_test.go`
- Modify: `src/app/tools/di.go` — register tool

### Steps

- [x] Define getWellnessTrendArgs struct with Oldest (required), Newest (required) fields
- [x] Implement NewGetWellnessTrendTool(client) ToolRegistration that calls GET /api/v1/athlete/{id}/wellness with oldest/newest query params
- [x] Return response JSON as TextContent
- [x] Register in tools.Module
- [x] Write tests
- [x] Run `go test ./src/...` — must pass before verification

## Verification

- [x] Run full test suite: `go test ./src/...`
- [x] Run linter: `golangci-lint run ./src/...`

## Wrap Up

- [x] Update CLAUDE.md if internal patterns changed
- [x] Move this plan to `docs/plans/completed/`
