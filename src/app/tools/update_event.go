package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/clients/intervals"
)

type updateEventArgs struct {
	EventID        string   `json:"event_id"                   jsonschema:"the event ID to update"`
	Name           string   `json:"name,omitempty"             jsonschema:"event name"`
	Description    string   `json:"description,omitempty"      jsonschema:"event description or notes"`
	StartDateLocal string   `json:"start_date_local,omitempty" jsonschema:"start date in yyyy-MM-dd format"`
	Category       string   `json:"category,omitempty"         jsonschema:"event category (WORKOUT, NOTE, RACE)"`
	Type           string   `json:"type,omitempty"             jsonschema:"sport type (e.g. Ride or Run or Swim)"`
	MovingTime     *float64 `json:"moving_time,omitempty"      jsonschema:"planned moving time in seconds"`
	Distance       *float64 `json:"distance,omitempty"         jsonschema:"planned distance in meters"`
	TrainingLoad   *float64 `json:"training_load,omitempty"    jsonschema:"planned training load (TSS or similar)"`
}

type updateEventPayload struct {
	Name           string   `json:"name,omitempty"`
	Description    string   `json:"description,omitempty"`
	StartDateLocal string   `json:"start_date_local,omitempty"`
	Category       string   `json:"category,omitempty"`
	Type           string   `json:"type,omitempty"`
	MovingTime     *float64 `json:"moving_time,omitempty"`
	Distance       *float64 `json:"distance,omitempty"`
	TrainingLoad   *float64 `json:"training_load,omitempty"`
}

// NewUpdateEventTool returns a ToolRegistration that registers the update_event tool on an MCP server.
func NewUpdateEventTool(apiClient *intervals.Client) ToolRegistration {
	return func(server *mcp.Server) {
		mcp.AddTool(server,
			&mcp.Tool{
				Name:        "update_event",
				Description: "Updates an existing event for the athlete. Only the provided fields will be updated.",
			},
			func(ctx context.Context, _ *mcp.CallToolRequest, args updateEventArgs) (*mcp.CallToolResult, any, error) {
				if args.EventID == "" {
					return nil, nil, errMissingEventID
				}

				if args.StartDateLocal != "" {
					err := validateDateFormat(args.StartDateLocal)
					if err != nil {
						return nil, nil, err
					}
				}

				payload := updateEventPayload{
					Name:           args.Name,
					Description:    args.Description,
					StartDateLocal: args.StartDateLocal,
					Category:       args.Category,
					Type:           args.Type,
					MovingTime:     args.MovingTime,
					Distance:       args.Distance,
					TrainingLoad:   args.TrainingLoad,
				}

				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					return nil, nil, fmt.Errorf("marshaling event body: %w", err)
				}

				path := "/api/v1/athlete/" + apiClient.AthleteID() + "/events/" + url.PathEscape(args.EventID)

				body, err := apiClient.Put(ctx, path, bytes.NewReader(payloadBytes))
				if err != nil {
					return nil, nil, fmt.Errorf("updating event: %w", err)
				}

				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: string(body),
						},
					},
				}, nil, nil
			},
		)
	}
}
