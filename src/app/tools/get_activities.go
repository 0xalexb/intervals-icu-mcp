package tools

import (
	"context"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/clients/intervals"
)

type getActivitiesArgs struct {
	Oldest string `json:"oldest"           jsonschema:"oldest date in yyyy-MM-dd format (inclusive)"`
	Newest string `json:"newest,omitempty" jsonschema:"newest date in yyyy-MM-dd format (inclusive), defaults to today"`
}

// NewGetActivitiesTool returns a ToolRegistration that registers the get_activities tool on an MCP server.
func NewGetActivitiesTool(apiClient *intervals.Client) ToolRegistration {
	return func(server *mcp.Server) {
		mcp.AddTool(server,
			&mcp.Tool{
				Name:        "get_activities",
				Description: "Returns a list of activities for the athlete within the specified date range.",
			},
			func(ctx context.Context, _ *mcp.CallToolRequest, args getActivitiesArgs) (*mcp.CallToolResult, any, error) {
				if args.Oldest == "" {
					return nil, nil, errMissingOldest
				}

				err := validateDateFormat(args.Oldest)
				if err != nil {
					return nil, nil, err
				}

				if args.Newest != "" {
					err = validateDateFormat(args.Newest)
					if err != nil {
						return nil, nil, err
					}
				}

				params := url.Values{}
				params.Set("oldest", args.Oldest)

				if args.Newest != "" {
					params.Set("newest", args.Newest)
				}

				body, err := apiClient.Get(ctx, "/api/v1/athlete/"+apiClient.AthleteID()+"/activities", params)
				if err != nil {
					return nil, nil, fmt.Errorf("fetching activities: %w", err)
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
