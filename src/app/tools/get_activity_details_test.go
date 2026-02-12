package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/client"
)

func TestNewGetActivityDetailsTool_Success(t *testing.T) {
	t.Parallel()

	const activityID = "i12345e678"
	const responseJSON = `{"id":"i12345e678","type":"Ride","start_date_local":"2024-01-15","distance":42000,"moving_time":3600}`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got: %s", r.Method)
		}

		if r.URL.Path != "/api/v1/activity/"+activityID {
			t.Errorf("expected path /api/v1/activity/%s, got: %s", activityID, r.URL.Path)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != "API_KEY" || pass != "test-key" {
			t.Errorf("expected basic auth API_KEY:test-key, got: %s:%s (ok=%v)", user, pass, ok)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer apiServer.Close()

	c := client.NewTestClient(apiServer.URL, "test-key", "i12345", apiServer.Client())

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "test-server",
			Version: "v0.0.1",
		},
		nil,
	)

	registration := NewGetActivityDetailsTool(c)
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

	mcpClient := mcp.NewClient(
		&mcp.Implementation{
			Name:    "test-client",
			Version: "v0.0.1",
		},
		nil,
	)

	connectCtx, connectCancel := context.WithTimeout(ctx, 5*time.Second)
	defer connectCancel()

	session, err := mcpClient.Connect(connectCtx, clientTransport, nil)
	if err != nil {
		t.Fatalf("expected client to connect, got: %v", err)
	}

	t.Cleanup(func() { _ = session.Close() })

	callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
	defer callCancel()

	result, err := session.CallTool(callCtx, &mcp.CallToolParams{
		Name: "get_activity_details",
		Arguments: map[string]any{
			"activity_id": activityID,
		},
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

	if textContent.Text != responseJSON {
		t.Fatalf("expected text %q, got: %q", responseJSON, textContent.Text)
	}
}

func TestNewGetActivityDetailsTool_APIError(t *testing.T) {
	t.Parallel()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"activity not found"}`))
	}))
	defer apiServer.Close()

	c := client.NewTestClient(apiServer.URL, "test-key", "i12345", apiServer.Client())

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "test-server",
			Version: "v0.0.1",
		},
		nil,
	)

	registration := NewGetActivityDetailsTool(c)
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

	mcpClient := mcp.NewClient(
		&mcp.Implementation{
			Name:    "test-client",
			Version: "v0.0.1",
		},
		nil,
	)

	connectCtx, connectCancel := context.WithTimeout(ctx, 5*time.Second)
	defer connectCancel()

	session, err := mcpClient.Connect(connectCtx, clientTransport, nil)
	if err != nil {
		t.Fatalf("expected client to connect, got: %v", err)
	}

	t.Cleanup(func() { _ = session.Close() })

	callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
	defer callCancel()

	result, err := session.CallTool(callCtx, &mcp.CallToolParams{
		Name: "get_activity_details",
		Arguments: map[string]any{
			"activity_id": "nonexistent",
		},
	})
	if err != nil {
		t.Fatalf("expected CallTool to succeed (error returned in result), got: %v", err)
	}

	if result.IsError != true {
		t.Fatalf("expected IsError to be true, got: %v", result.IsError)
	}
}
