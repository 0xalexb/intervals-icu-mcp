package tools

import (
	"context"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/clients/intervals"
)

type getPowerCurveArgs struct {
	Type   string `json:"type"             jsonschema:"activity type, e.g. Ride, Run, Swim"`
	Newest string `json:"newest,omitempty" jsonschema:"newest date in yyyy-MM-dd format (inclusive), defaults to today"`
}

// NewGetPowerCurveTool returns a ToolRegistration that registers the get_power_curve tool on an MCP server.
func NewGetPowerCurveTool(apiClient *intervals.Client) ToolRegistration {
	return func(server *mcp.Server) {
		mcp.AddTool(server,
			&mcp.Tool{
				Name:        "get_power_curve",
				Description: "Returns the power curve data for the athlete filtered by activity type.",
			},
			func(ctx context.Context, _ *mcp.CallToolRequest, args getPowerCurveArgs) (*mcp.CallToolResult, any, error) {
				if args.Type == "" {
					return nil, nil, errMissingType
				}

				if args.Newest != "" {
					err := validateDateFormat(args.Newest)
					if err != nil {
						return nil, nil, err
					}
				}

				params := url.Values{}
				params.Set("type", args.Type)

				if args.Newest != "" {
					params.Set("newest", args.Newest)
				}

				body, err := apiClient.Get(ctx, "/api/v1/athlete/"+apiClient.AthleteID()+"/power-curves.json", params)
				if err != nil {
					return nil, nil, fmt.Errorf("fetching power curve: %w", err)
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
