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

	"github.com/0xalexb/intervals-icu-mcp/src/app/clients/intervals"
)

func TestNewCreateEventTool_Success(t *testing.T) {
	t.Parallel()

	const athleteID = "i12345"
	const responseJSON = `{"id":42,"category":"WORKOUT","name":"Morning Run","start_date_local":"2024-03-15"}`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got: %s", r.Method)
		}

		if r.URL.Path != "/api/v1/athlete/"+athleteID+"/events" {
			t.Errorf("expected path /api/v1/athlete/%s/events, got: %s", athleteID, r.URL.Path)
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

		if parsed["name"] != "Morning Run" {
			t.Errorf("expected name=Morning Run, got: %v", parsed["name"])
		}

		if parsed["start_date_local"] != "2024-03-15" {
			t.Errorf("expected start_date_local=2024-03-15, got: %v", parsed["start_date_local"])
		}

		if parsed["category"] != "WORKOUT" {
			t.Errorf("expected category=WORKOUT, got: %v", parsed["category"])
		}

		w.WriteHeader(http.StatusCreated)
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

	registration := NewCreateEventTool(c)
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
		Name: "create_event",
		Arguments: map[string]any{
			"name":             "Morning Run",
			"start_date_local": "2024-03-15",
			"category":         "WORKOUT",
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

func TestNewCreateEventTool_WithOptionalFields(t *testing.T) {
	t.Parallel()

	const athleteID = "i12345"
	const responseJSON = `{"id":43,"category":"WORKOUT","name":"Long Ride"}`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody, _ := io.ReadAll(r.Body)

		var parsed map[string]any
		err := json.Unmarshal(reqBody, &parsed)
		if err != nil {
			t.Fatalf("expected valid JSON body, got error: %v", err)
		}

		if parsed["name"] != "Long Ride" {
			t.Errorf("expected name=Long Ride, got: %v", parsed["name"])
		}

		if parsed["type"] != "Ride" {
			t.Errorf("expected type=Ride, got: %v", parsed["type"])
		}

		if parsed["description"] != "Endurance ride" {
			t.Errorf("expected description=Endurance ride, got: %v", parsed["description"])
		}

		w.WriteHeader(http.StatusCreated)
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

	registration := NewCreateEventTool(c)
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
		Name: "create_event",
		Arguments: map[string]any{
			"name":             "Long Ride",
			"start_date_local": "2024-03-20",
			"category":         "WORKOUT",
			"type":             "Ride",
			"description":      "Endurance ride",
			"moving_time":      7200.0,
			"distance":         80000.0,
			"training_load":    150.0,
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

func TestNewCreateEventTool_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args map[string]any
	}{
		{
			name: "empty name",
			args: map[string]any{
				"name":             "",
				"start_date_local": "2024-03-15",
				"category":         "WORKOUT",
			},
		},
		{
			name: "empty start_date_local",
			args: map[string]any{
				"name":             "Morning Run",
				"start_date_local": "",
				"category":         "WORKOUT",
			},
		},
		{
			name: "empty category",
			args: map[string]any{
				"name":             "Morning Run",
				"start_date_local": "2024-03-15",
				"category":         "",
			},
		},
		{
			name: "invalid date format",
			args: map[string]any{
				"name":             "Morning Run",
				"start_date_local": "15-03-2024",
				"category":         "WORKOUT",
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

			registration := NewCreateEventTool(c)
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
				Name:      "create_event",
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

func TestNewCreateEventTool_APIError(t *testing.T) {
	t.Parallel()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid event data"}`))
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

	registration := NewCreateEventTool(c)
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
		Name: "create_event",
		Arguments: map[string]any{
			"name":             "Bad Event",
			"start_date_local": "2024-03-15",
			"category":         "INVALID",
		},
	})
	if err != nil {
		t.Fatalf("expected CallTool to succeed (error returned in result), got: %v", err)
	}

	if result.IsError != true {
		t.Fatalf("expected IsError to be true, got: %v", result.IsError)
	}
}
