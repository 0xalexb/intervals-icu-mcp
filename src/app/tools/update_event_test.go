package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/client"
)

func TestNewUpdateEventTool_Success(t *testing.T) {
	t.Parallel()

	const athleteID = "i12345"
	const eventID = "e99"
	const responseJSON = `{"id":99,"category":"WORKOUT","name":"Updated Run","start_date_local":"2024-03-16"}`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got: %s", r.Method)
		}

		if r.URL.Path != "/api/v1/athlete/"+athleteID+"/events/"+eventID {
			t.Errorf("expected path /api/v1/athlete/%s/events/%s, got: %s", athleteID, eventID, r.URL.Path)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != "API_KEY" || pass != "test-key" {
			t.Errorf("expected basic auth API_KEY:test-key, got: %s:%s (ok=%v)", user, pass, ok)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got: %s", r.Header.Get("Content-Type"))
		}

		reqBody, _ := io.ReadAll(r.Body)

		var parsed map[string]any
		err := json.Unmarshal(reqBody, &parsed)
		if err != nil {
			t.Fatalf("expected valid JSON body, got error: %v", err)
		}

		if parsed["name"] != "Updated Run" {
			t.Errorf("expected name=Updated Run, got: %v", parsed["name"])
		}

		if parsed["start_date_local"] != "2024-03-16" {
			t.Errorf("expected start_date_local=2024-03-16, got: %v", parsed["start_date_local"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer apiServer.Close()

	c := client.NewTestClient(apiServer.URL, "test-key", athleteID, apiServer.Client())

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "test-server",
			Version: "v0.0.1",
		},
		nil,
	)

	registration := NewUpdateEventTool(c)
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
		Name: "update_event",
		Arguments: map[string]any{
			"event_id":         eventID,
			"name":             "Updated Run",
			"start_date_local": "2024-03-16",
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

func TestNewUpdateEventTool_WithOptionalFields(t *testing.T) {
	t.Parallel()

	const athleteID = "i12345"
	const eventID = "e88"
	const responseJSON = `{"id":88,"category":"WORKOUT","name":"Long Ride Updated"}`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody, _ := io.ReadAll(r.Body)

		var parsed map[string]any
		err := json.Unmarshal(reqBody, &parsed)
		if err != nil {
			t.Fatalf("expected valid JSON body, got error: %v", err)
		}

		if parsed["name"] != "Long Ride Updated" {
			t.Errorf("expected name=Long Ride Updated, got: %v", parsed["name"])
		}

		if parsed["type"] != "Ride" {
			t.Errorf("expected type=Ride, got: %v", parsed["type"])
		}

		if parsed["description"] != "Updated endurance ride" {
			t.Errorf("expected description=Updated endurance ride, got: %v", parsed["description"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer apiServer.Close()

	c := client.NewTestClient(apiServer.URL, "test-key", athleteID, apiServer.Client())

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "test-server",
			Version: "v0.0.1",
		},
		nil,
	)

	registration := NewUpdateEventTool(c)
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
		Name: "update_event",
		Arguments: map[string]any{
			"event_id":      eventID,
			"name":          "Long Ride Updated",
			"type":          "Ride",
			"description":   "Updated endurance ride",
			"moving_time":   7200.0,
			"distance":      80000.0,
			"training_load": 150.0,
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

func TestNewUpdateEventTool_Validation(t *testing.T) {
	t.Parallel()

	apiServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("API should not be called for validation errors")
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

	registration := NewUpdateEventTool(c)
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
		Name: "update_event",
		Arguments: map[string]any{
			"event_id":         "e99",
			"start_date_local": "16-03-2024",
		},
	})
	if err != nil {
		t.Fatalf("expected CallTool to succeed (error returned in result), got: %v", err)
	}

	if result.IsError != true {
		t.Fatalf("expected IsError to be true, got: %v", result.IsError)
	}
}

func TestNewUpdateEventTool_APIError(t *testing.T) {
	t.Parallel()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"event not found"}`))
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

	registration := NewUpdateEventTool(c)
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
		Name: "update_event",
		Arguments: map[string]any{
			"event_id": "nonexistent",
			"name":     "Updated Name",
		},
	})
	if err != nil {
		t.Fatalf("expected CallTool to succeed (error returned in result), got: %v", err)
	}

	if result.IsError != true {
		t.Fatalf("expected IsError to be true, got: %v", result.IsError)
	}
}
