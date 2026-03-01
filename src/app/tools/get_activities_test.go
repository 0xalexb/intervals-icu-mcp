package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xalexb/intervals-icu-mcp/src/app/clients/intervals"
)

func TestNewGetActivitiesTool_Success(t *testing.T) {
	t.Parallel()

	const athleteID = "i12345"
	const responseJSON = `[{"id":"a1","type":"Ride","start_date_local":"2024-01-15"}]`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got: %s", r.Method)
		}

		if r.URL.Path != "/api/v1/athlete/"+athleteID+"/activities" {
			t.Errorf("expected path /api/v1/athlete/%s/activities, got: %s", athleteID, r.URL.Path)
		}

		oldest := r.URL.Query().Get("oldest")
		if oldest != "2024-01-01" {
			t.Errorf("expected oldest=2024-01-01, got: %s", oldest)
		}

		newest := r.URL.Query().Get("newest")
		if newest != "2024-01-31" {
			t.Errorf("expected newest=2024-01-31, got: %s", newest)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != "API_KEY" || pass != "test-key" {
			t.Errorf("expected basic auth API_KEY:test-key, got: %s:%s (ok=%v)", user, pass, ok)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer apiServer.Close()

	c := intervals.NewTestClient(apiServer.URL, "test-key", athleteID, apiServer.Client())

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "test-server",
			Version: "v0.0.1",
		},
		nil,
	)

	registration := NewGetActivitiesTool(c)
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
		Name: "get_activities",
		Arguments: map[string]any{
			"oldest": "2024-01-01",
			"newest": "2024-01-31",
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

func TestNewGetActivitiesTool_WithoutNewest(t *testing.T) {
	t.Parallel()

	const athleteID = "i12345"
	const responseJSON = `[{"id":"a1","type":"Ride"}]`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oldest := r.URL.Query().Get("oldest")
		if oldest != "2024-01-01" {
			t.Errorf("expected oldest=2024-01-01, got: %s", oldest)
		}

		newest := r.URL.Query().Get("newest")
		if newest != "" {
			t.Errorf("expected no newest param, got: %s", newest)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer apiServer.Close()

	c := intervals.NewTestClient(apiServer.URL, "test-key", athleteID, apiServer.Client())

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "test-server",
			Version: "v0.0.1",
		},
		nil,
	)

	registration := NewGetActivitiesTool(c)
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
		Name: "get_activities",
		Arguments: map[string]any{
			"oldest": "2024-01-01",
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

func TestNewGetActivitiesTool_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args map[string]any
	}{
		{
			name: "invalid oldest date format",
			args: map[string]any{
				"oldest": "01-01-2024",
			},
		},
		{
			name: "invalid newest date format",
			args: map[string]any{
				"oldest": "2024-01-01",
				"newest": "31/01/2024",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			apiServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				t.Fatal("API should not be called for validation errors")
			}))
			defer apiServer.Close()

			c := intervals.NewTestClient(apiServer.URL, "test-key", "i12345", apiServer.Client())

			mcpServer := mcp.NewServer(
				&mcp.Implementation{
					Name:    "test-server",
					Version: "v0.0.1",
				},
				nil,
			)

			registration := NewGetActivitiesTool(c)
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
				Name:      "get_activities",
				Arguments: tt.args,
			})
			if err != nil {
				t.Fatalf("expected CallTool to succeed (error returned in result), got: %v", err)
			}

			if result.IsError != true {
				t.Fatalf("expected IsError to be true, got: %v", result.IsError)
			}
		})
	}
}

func TestNewGetActivitiesTool_APIError(t *testing.T) {
	t.Parallel()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer apiServer.Close()

	c := intervals.NewTestClient(apiServer.URL, "test-key", "i12345", apiServer.Client())

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "test-server",
			Version: "v0.0.1",
		},
		nil,
	)

	registration := NewGetActivitiesTool(c)
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
		Name: "get_activities",
		Arguments: map[string]any{
			"oldest": "2024-01-01",
		},
	})
	if err != nil {
		t.Fatalf("expected CallTool to succeed (error returned in result), got: %v", err)
	}

	if result.IsError != true {
		t.Fatalf("expected IsError to be true, got: %v", result.IsError)
	}
}
