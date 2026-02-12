package tools

import "go.uber.org/fx"

// Module provides all MCP tool constructors via the "mcp_tools" fx group.
var Module = fx.Module("tools", //nolint:gochecknoglobals // fx.Module as package variable is the standard DI pattern.
	fx.Provide(fx.Annotate(NewVersionTool, fx.ResultTags(`group:"mcp_tools"`))),
	fx.Provide(fx.Annotate(NewGetAthleteProfileTool, fx.ResultTags(`group:"mcp_tools"`))),
	fx.Provide(fx.Annotate(NewGetActivitiesTool, fx.ResultTags(`group:"mcp_tools"`))),
	fx.Provide(fx.Annotate(NewGetActivityDetailsTool, fx.ResultTags(`group:"mcp_tools"`))),
	fx.Provide(fx.Annotate(NewGetEventsTool, fx.ResultTags(`group:"mcp_tools"`))),
	fx.Provide(fx.Annotate(NewCreateEventTool, fx.ResultTags(`group:"mcp_tools"`))),
	fx.Provide(fx.Annotate(NewUpdateEventTool, fx.ResultTags(`group:"mcp_tools"`))),
	fx.Provide(fx.Annotate(NewDeleteEventTool, fx.ResultTags(`group:"mcp_tools"`))),
	fx.Provide(fx.Annotate(NewGetPowerCurveTool, fx.ResultTags(`group:"mcp_tools"`))),
	fx.Provide(fx.Annotate(NewGetWellnessTool, fx.ResultTags(`group:"mcp_tools"`))),
	fx.Provide(fx.Annotate(NewGetWellnessTrendTool, fx.ResultTags(`group:"mcp_tools"`))),
)
