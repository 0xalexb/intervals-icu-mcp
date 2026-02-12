package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/tools"
)

func TestNewServer_Stdio(t *testing.T) {
	t.Parallel()

	server := NewServer(TransportStdio, []tools.ToolRegistration{tools.NewVersionTool()})

	if server == nil {
		t.Fatal("expected non-nil server")
	}

	if server.mcpServer == nil {
		t.Fatal("expected non-nil mcp server")
	}

	if server.stdioTransport == nil {
		t.Fatal("expected non-nil transport for stdio mode")
	}
}

func TestNewServer_Streamable(t *testing.T) {
	t.Parallel()

	server := NewServer(TransportStreamable, []tools.ToolRegistration{tools.NewVersionTool()})

	if server == nil {
		t.Fatal("expected non-nil server")
	}

	if server.mcpServer == nil {
		t.Fatal("expected non-nil mcp server")
	}

	if server.stdioTransport != nil {
		t.Fatal("expected nil transport for streamable mode")
	}
}

func TestServer_StartStop_Stdio(t *testing.T) {
	t.Parallel()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := NewServer(TransportStdio, []tools.ToolRegistration{tools.NewVersionTool()})
	server.stdioTransport = serverTransport

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

func TestServer_StartStop_Streamable(t *testing.T) {
	t.Parallel()

	server := NewServer(TransportStreamable, []tools.ToolRegistration{tools.NewVersionTool()})

	ctx := context.Background()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("expected no error on start, got: %v", err)
	}

	err = server.Stop(ctx)
	if err != nil {
		t.Fatalf("expected no error on stop, got: %v", err)
	}
}

func TestServer_Handler(t *testing.T) {
	t.Parallel()

	server := NewServer(TransportStreamable, []tools.ToolRegistration{tools.NewVersionTool()})

	handler := server.Handler()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestServer_Streamable_Responds(t *testing.T) {
	t.Parallel()

	server := NewServer(TransportStreamable, []tools.ToolRegistration{tools.NewVersionTool()})

	handler := server.Handler()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}`

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/mcp", strings.NewReader(body))
	if err != nil {
		t.Fatalf("expected no error creating request, got: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestServer_StopWithoutStart(t *testing.T) {
	t.Parallel()

	server := NewServer(TransportStdio, []tools.ToolRegistration{tools.NewVersionTool()})

	err := server.Stop(context.Background())
	if err != nil {
		t.Fatalf("expected no error stopping unstarted server, got: %v", err)
	}
}

func TestServer_DoubleStart_Stdio(t *testing.T) {
	t.Parallel()

	serverTransport, _ := mcp.NewInMemoryTransports()
	server := NewServer(TransportStdio, []tools.ToolRegistration{tools.NewVersionTool()})
	server.stdioTransport = serverTransport

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

func TestServer_DoubleStart_Streamable(t *testing.T) {
	t.Parallel()

	server := NewServer(TransportStreamable, []tools.ToolRegistration{tools.NewVersionTool()})

	ctx := context.Background()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("expected no error on first start, got: %v", err)
	}

	// Streamable Start is a no-op, so calling it again should also succeed.
	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("expected no error on second start (streamable is no-op), got: %v", err)
	}
}
