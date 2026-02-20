package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	di "github.com/0xalexb/hjarta-di"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/tools"
)

const sessionIdleTimeout = 30 * time.Minute

var (
	errAlreadyStarted      = errors.New("server already started")
	errUnsupportedTransport = errors.New("unsupported transport")
)

// Server wraps the MCP server with lifecycle management.
// For stdio transport, Start/Stop manage the server goroutine.
// For streamable transport, Start/Stop are no-ops since hjarta-di manages the HTTP lifecycle.
type Server struct {
	mcpServer      *mcp.Server
	stdioTransport mcp.Transport
	transport      Transport
	cancel         context.CancelFunc
	done           chan struct{}
}

// NewServer creates a new MCP server instance.
func NewServer(transport Transport, toolRegistrations []tools.ToolRegistration) *Server {
	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "intervals-icu-mcp",
			Title:   "Intervals ICU MCP Server",
			Version: di.Version,
		},
		nil,
	)

	for _, register := range toolRegistrations {
		register(mcpServer)
	}

	srv := &Server{
		mcpServer: mcpServer,
		transport: transport,
	}

	if transport == TransportStdio {
		srv.stdioTransport = &mcp.StdioTransport{}
	}

	return srv
}

// Handler returns the http.Handler for the streamable transport.
func (s *Server) Handler() http.Handler {
	return mcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *mcp.Server {
			return s.mcpServer
		},
		&mcp.StreamableHTTPOptions{
			SessionTimeout: sessionIdleTimeout,
		},
	)
}

// Start launches the MCP server in a background goroutine.
// For streamable transport, this is a no-op since hjarta-di manages the HTTP lifecycle.
//
//nolint:contextcheck // fx.Hook requires this signature; server uses its own context.
func (s *Server) Start(_ context.Context) error {
	if s.transport == TransportStreamable {
		return nil
	}

	if s.transport != TransportStdio {
		return fmt.Errorf("%w: %q", errUnsupportedTransport, s.transport)
	}

	if s.done != nil {
		return errAlreadyStarted
	}

	s.done = make(chan struct{})

	return s.startStdio()
}

// Stop cancels the server context and waits for the server goroutine to finish.
func (s *Server) Stop(ctx context.Context) error {
	if s.transport == TransportStreamable || s.done == nil {
		return nil
	}

	slog.Info("Stopping MCP server")

	if s.cancel != nil {
		s.cancel()
	}

	select {
	case <-s.done:
	case <-ctx.Done():
		return fmt.Errorf("waiting for server to stop: %w", ctx.Err())
	}

	return nil
}

func (s *Server) startStdio() error {
	serverCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	go func() {
		defer close(s.done)

		slog.Info("Starting MCP server", "transport", "stdio")

		err := s.mcpServer.Run(serverCtx, s.stdioTransport)
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("MCP server error", "error", err)
		}
	}()

	return nil
}
