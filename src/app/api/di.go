package api

import (
	"go.uber.org/fx"
)

// Module provides the HTTP router with the MCP handler mounted at /mcp.
var Module = fx.Module("api", //nolint:gochecknoglobals // fx.Module as package variable is the standard DI pattern.
	fx.Provide(fx.Annotate(
		NewRouter,
		fx.ParamTags(`name:"mcp-raw"`),
		fx.ResultTags(`name:"mcp"`),
	)),
)
