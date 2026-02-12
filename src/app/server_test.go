package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/tools"
)

func TestNewServer(t *testing.T) {
	t.Parallel()

	transport := &mcp.StdioTransport{}
	server := NewServer(transport, []tools.ToolRegistration{tools.NewVersionTool()})

	if server == nil {
		t.Fatal("expected non-nil server")
	}

	if server.mcpServer == nil {
		t.Fatal("expected non-nil mcp server")
	}

	if server.transport == nil {
		t.Fatal("expected non-nil transport")
	}
}

func TestServer_StartStop(t *testing.T) {
	t.Parallel()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := NewServer(serverTransport, []tools.ToolRegistration{tools.NewVersionTool()})

	ctx := context.Background()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("expected no error on start, got: %v", err)
	}

	t.Cleanup(func() { _ = server.Stop(ctx) })

	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "test-client",
			Version: "v0.0.1",
		},
		nil,
	)

	connectCtx, connectCancel := context.WithTimeout(ctx, 5*time.Second)
	defer connectCancel()

	session, err := client.Connect(connectCtx, clientTransport, nil)
	if err != nil {
		t.Fatalf("expected client to connect, got: %v", err)
	}

	t.Cleanup(func() { _ = session.Close() })

	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()

	err = session.Ping(pingCtx, nil)
	if err != nil {
		t.Fatalf("expected ping to succeed, got: %v", err)
	}
}

func TestServer_StopWithoutStart(t *testing.T) {
	t.Parallel()

	server := NewServer(&mcp.StdioTransport{}, []tools.ToolRegistration{tools.NewVersionTool()})

	err := server.Stop(context.Background())
	if err != nil {
		t.Fatalf("expected no error stopping unstarted server, got: %v", err)
	}
}

func TestServer_DoubleStart(t *testing.T) {
	t.Parallel()

	serverTransport, _ := mcp.NewInMemoryTransports()
	server := NewServer(serverTransport, []tools.ToolRegistration{tools.NewVersionTool()})

	ctx := context.Background()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("expected no error on first start, got: %v", err)
	}

	t.Cleanup(func() { _ = server.Stop(ctx) })

	err = server.Start(ctx)
	if err == nil {
		t.Fatal("expected error on second start")
	}

	if !errors.Is(err, errAlreadyStarted) {
		t.Fatalf("expected errAlreadyStarted, got: %v", err)
	}
}
