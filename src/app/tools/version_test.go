package tools

import (
	"context"
	"fmt"
	"testing"
	"time"

	di "github.com/0xalexb/hjarta-di"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewVersionTool_RegistersCallableTool(t *testing.T) {
	t.Parallel()

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "test-server",
			Version: "v0.0.1",
		},
		nil,
	)

	registration := NewVersionTool()
	registration(mcpServer)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()

	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	serverDone := make(chan struct{})

	go func() {
		defer close(serverDone)

		_ = mcpServer.Run(serverCtx, serverTransport)
	}()

	t.Cleanup(func() {
		serverCancel()
		<-serverDone
	})

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

	callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
	defer callCancel()

	result, err := session.CallTool(callCtx, &mcp.CallToolParams{
		Name: "version",
	})
	if err != nil {
		t.Fatalf("expected CallTool to succeed, got: %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got: %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got: %T", result.Content[0])
	}

	expectedText := fmt.Sprintf("Version: %s\nDI Version: %s\nCompiled At: %s", di.Version, di.DIVersion, di.CompiledAt)
	if textContent.Text != expectedText {
		t.Fatalf("expected text %q, got: %q", expectedText, textContent.Text)
	}
}
