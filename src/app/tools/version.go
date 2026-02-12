package tools

import (
	"context"
	"fmt"

	di "github.com/0xalexb/hjarta-di"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type versionArgs struct{}

// NewVersionTool returns a ToolRegistration that registers the version tool on an MCP server.
func NewVersionTool() ToolRegistration {
	return func(server *mcp.Server) {
		mcp.AddTool(server,
			&mcp.Tool{
				Name:        "version",
				Description: "Returns application build information including version, DI version, and compilation time.",
			},
			func(_ context.Context, _ *mcp.CallToolRequest, _ versionArgs) (*mcp.CallToolResult, any, error) {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: fmt.Sprintf("Version: %s\nDI Version: %s\nCompiled At: %s", di.Version, di.DIVersion, di.CompiledAt),
						},
					},
				}, nil, nil
			},
		)
	}
}
