package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/clients/intervals"
)

type getAthleteProfileArgs struct{}

// NewGetAthleteProfileTool returns a ToolRegistration for the get_athlete_profile tool.
func NewGetAthleteProfileTool(apiClient *intervals.Client) ToolRegistration {
	return func(server *mcp.Server) {
		mcp.AddTool(server,
			&mcp.Tool{
				Name: "get_athlete_profile",
				Description: "Returns the athlete profile from Intervals.icu, " +
					"including settings, sport info, and current fitness data.",
			},
			func(
				ctx context.Context, _ *mcp.CallToolRequest, _ getAthleteProfileArgs,
			) (*mcp.CallToolResult, any, error) {
				body, err := apiClient.Get(ctx, "/api/v1/athlete/"+apiClient.AthleteID(), nil)
				if err != nil {
					return nil, nil, fmt.Errorf("fetching athlete profile: %w", err)
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
