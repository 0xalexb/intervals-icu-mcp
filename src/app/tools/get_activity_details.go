package tools

import (
	"context"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/client"
)

type getActivityDetailsArgs struct {
	ActivityID string `json:"activity_id" jsonschema:"activity ID to retrieve details for"`
}

// NewGetActivityDetailsTool returns a ToolRegistration for the get_activity_details tool.
func NewGetActivityDetailsTool(apiClient *client.Client) ToolRegistration {
	return func(server *mcp.Server) {
		mcp.AddTool(server,
			&mcp.Tool{
				Name:        "get_activity_details",
				Description: "Returns detailed information about a specific activity from Intervals.icu.",
			},
			func(ctx context.Context, _ *mcp.CallToolRequest, args getActivityDetailsArgs) (*mcp.CallToolResult, any, error) {
				if args.ActivityID == "" {
					return nil, nil, errMissingActivityID
				}

				body, err := apiClient.Get(ctx, "/api/v1/activity/"+url.PathEscape(args.ActivityID), nil)
				if err != nil {
					return nil, nil, fmt.Errorf("fetching activity details: %w", err)
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
