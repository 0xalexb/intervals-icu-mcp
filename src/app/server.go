package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	di "github.com/0xalexb/hjarta-di"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/tools"
)

var errAlreadyStarted = errors.New("server already started")

// Server wraps the MCP server with lifecycle management.
type Server struct {
	mcpServer *mcp.Server
	transport mcp.Transport
	cancel    context.CancelFunc
	done      chan struct{}
}

// NewServer creates a new MCP server instance.
func NewServer(transport mcp.Transport, toolRegistrations []tools.ToolRegistration) *Server {
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

	return &Server{
		mcpServer: mcpServer,
		transport: transport,
	}
}

// Start launches the MCP server in a background goroutine.
//nolint:contextcheck // fx.Hook requires this signature; server uses its own context.
func (s *Server) Start(_ context.Context) error {
	if s.done != nil {
		return errAlreadyStarted
	}

	serverCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})

	go func() {
		defer close(s.done)

		slog.Info("Starting MCP server")

		err := s.mcpServer.Run(serverCtx, s.transport)
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("MCP server error", "error", err)
		}
	}()

	return nil
}

// Stop cancels the server context and waits for the server goroutine to finish.
func (s *Server) Stop(ctx context.Context) error {
	if s.cancel != nil {
		slog.Info("Stopping MCP server")
		s.cancel()

		select {
		case <-s.done:
		case <-ctx.Done():
			return fmt.Errorf("waiting for server to stop: %w", ctx.Err())
		}
	}

	return nil
}
