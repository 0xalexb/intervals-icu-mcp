package app

// Transport represents the transport mode for the MCP server.
type Transport string

const (
	// TransportStdio selects the stdio transport mode.
	TransportStdio Transport = "stdio"
	// TransportStreamable selects the streamable HTTP transport mode.
	TransportStreamable Transport = "streamable"
)
