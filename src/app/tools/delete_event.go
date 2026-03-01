package tools

import (
	"context"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/clients/intervals"
)

type deleteEventArgs struct {
	EventID string `json:"event_id" jsonschema:"the event ID to delete"`
}

// NewDeleteEventTool returns a ToolRegistration that registers the delete_event tool on an MCP server.
func NewDeleteEventTool(apiClient *intervals.Client) ToolRegistration {
	return func(server *mcp.Server) {
		mcp.AddTool(server,
			&mcp.Tool{
				Name:        "delete_event",
				Description: "Deletes an event for the athlete by event ID.",
			},
			func(ctx context.Context, _ *mcp.CallToolRequest, args deleteEventArgs) (*mcp.CallToolResult, any, error) {
				if args.EventID == "" {
					return nil, nil, errMissingEventID
				}

				path := "/api/v1/athlete/" + apiClient.AthleteID() + "/events/" + url.PathEscape(args.EventID)

				_, err := apiClient.Delete(ctx, path, nil)
				if err != nil {
					return nil, nil, fmt.Errorf("deleting event: %w", err)
				}

				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: "Event " + args.EventID + " deleted successfully",
						},
					},
				}, nil, nil
			},
		)
	}
}
