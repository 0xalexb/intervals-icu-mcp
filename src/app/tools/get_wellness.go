package tools

import (
	"context"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/clients/intervals"
)

type getWellnessArgs struct {
	Date string `json:"date" jsonschema:"date in yyyy-MM-dd format"`
}

// NewGetWellnessTool returns a ToolRegistration that registers the get_wellness tool on an MCP server.
func NewGetWellnessTool(apiClient *intervals.Client) ToolRegistration {
	return func(server *mcp.Server) {
		mcp.AddTool(server,
			&mcp.Tool{
				Name:        "get_wellness",
				Description: "Returns wellness data for the athlete on a specific date.",
			},
			func(ctx context.Context, _ *mcp.CallToolRequest, args getWellnessArgs) (*mcp.CallToolResult, any, error) {
				if args.Date == "" {
					return nil, nil, errMissingDate
				}

				err := validateDateFormat(args.Date)
				if err != nil {
					return nil, nil, err
				}

				path := "/api/v1/athlete/" + apiClient.AthleteID() + "/wellness/" + url.PathEscape(args.Date)

				body, err := apiClient.Get(ctx, path, nil)
				if err != nil {
					return nil, nil, fmt.Errorf("fetching wellness: %w", err)
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
