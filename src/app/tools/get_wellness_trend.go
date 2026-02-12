package tools

import (
	"context"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/client"
)

type getWellnessTrendArgs struct {
	Oldest string `json:"oldest" jsonschema:"oldest date in yyyy-MM-dd format (inclusive)"`
	Newest string `json:"newest" jsonschema:"newest date in yyyy-MM-dd format (inclusive)"`
}

// NewGetWellnessTrendTool returns a ToolRegistration that registers the get_wellness_trend tool on an MCP server.
func NewGetWellnessTrendTool(apiClient *client.Client) ToolRegistration {
	return func(server *mcp.Server) {
		mcp.AddTool(server,
			&mcp.Tool{
				Name:        "get_wellness_trend",
				Description: "Returns wellness data for the athlete over a date range.",
			},
			func(ctx context.Context, _ *mcp.CallToolRequest, args getWellnessTrendArgs) (*mcp.CallToolResult, any, error) {
				if args.Oldest == "" {
					return nil, nil, errMissingOldest
				}

				if args.Newest == "" {
					return nil, nil, errMissingNewest
				}

				err := validateDateFormat(args.Oldest)
				if err != nil {
					return nil, nil, err
				}

				err = validateDateFormat(args.Newest)
				if err != nil {
					return nil, nil, err
				}

				params := url.Values{}
				params.Set("oldest", args.Oldest)
				params.Set("newest", args.Newest)

				body, err := apiClient.Get(ctx, "/api/v1/athlete/"+apiClient.AthleteID()+"/wellness", params)
				if err != nil {
					return nil, nil, fmt.Errorf("fetching wellness trend: %w", err)
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
