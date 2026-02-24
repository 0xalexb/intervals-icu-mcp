// Package app provides the MCP server, its DI module, and lifecycle management.
package app

import (
	"net/http"

	"go.uber.org/fx"

	"github.com/0xalexb/intervals-icu-mcp/src/app/api"
	"github.com/0xalexb/intervals-icu-mcp/src/app/auth"
	"github.com/0xalexb/intervals-icu-mcp/src/app/client"
	"github.com/0xalexb/intervals-icu-mcp/src/app/tools"
)

// Module provides the MCP server and its lifecycle hooks.
var Module = fx.Module("app", //nolint:gochecknoglobals // fx.Module as package variable is the standard DI pattern.
	api.Module,
	auth.Module,
	client.Module,
	tools.Module,
	fx.Provide(fx.Annotate(NewServer, fx.ParamTags(``, `group:"mcp_tools"`))),
	fx.Provide(fx.Annotate(
		func(server *Server) http.Handler { return server.Handler() },
		fx.ResultTags(`name:"mcp-raw"`),
	)),
	fx.Invoke(func(server *Server, lc fx.Lifecycle) {
		lc.Append(fx.Hook{
			OnStart: server.Start,
			OnStop:  server.Stop,
		})
	}),
)
