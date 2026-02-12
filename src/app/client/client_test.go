package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewClient_MissingAPIKey(t *testing.T) {
	t.Parallel()

	_, err := NewClient(Config{AthleteID: "i123"})
	if err != errMissingAPIKey {
		t.Fatalf("expected errMissingAPIKey, got: %v", err)
	}
}

func TestNewClient_MissingAthleteID(t *testing.T) {
	t.Parallel()

	_, err := NewClient(Config{APIKey: "some-key"})
	if err != errMissingAthleteID {
		t.Fatalf("expected errMissingAthleteID, got: %v", err)
	}
}

func TestNewClient_Success(t *testing.T) {
	t.Parallel()

	c, err := NewClient(Config{APIKey: "key123", AthleteID: "i456"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if c.AthleteID() != "i456" {
		t.Fatalf("expected athlete ID i456, got: %s", c.AthleteID())
	}
}

func TestClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got: %s", r.Method)
		}

		if r.URL.Path != "/api/v1/athlete/i123" {
			t.Errorf("expected path /api/v1/athlete/i123, got: %s", r.URL.Path)
		}

		if r.URL.Query().Get("oldest") != "2024-01-01" {
			t.Errorf("expected oldest=2024-01-01, got: %s", r.URL.Query().Get("oldest"))
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != "API_KEY" || pass != "test-key" {
			t.Errorf("expected basic auth API_KEY:test-key, got: %s:%s (ok=%v)", user, pass, ok)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"Test Athlete"}`))
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		apiKey:     "test-key",
		athleteID:  "i123",
		httpClient: server.Client(),
	}

	params := url.Values{"oldest": {"2024-01-01"}}
	body, err := c.Get(context.Background(), "/api/v1/athlete/i123", params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if string(body) != `{"name":"Test Athlete"}` {
		t.Fatalf("expected response body, got: %s", string(body))
	}
}

func TestClient_Post(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got: %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got: %s", r.Header.Get("Content-Type"))
		}

		reqBody, _ := io.ReadAll(r.Body)
		if string(reqBody) != `{"name":"event"}` {
			t.Errorf("expected body {\"name\":\"event\"}, got: %s", string(reqBody))
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"e1"}`))
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		apiKey:     "test-key",
		athleteID:  "i123",
		httpClient: server.Client(),
	}

	body, err := c.Post(context.Background(), "/api/v1/events", strings.NewReader(`{"name":"event"}`))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if string(body) != `{"id":"e1"}` {
		t.Fatalf("expected response body, got: %s", string(body))
	}
}

func TestClient_Put(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got: %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"updated":true}`))
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		apiKey:     "test-key",
		athleteID:  "i123",
		httpClient: server.Client(),
	}

	body, err := c.Put(context.Background(), "/api/v1/events/e1", strings.NewReader(`{"name":"updated"}`))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if string(body) != `{"updated":true}` {
		t.Fatalf("expected response body, got: %s", string(body))
	}
}

func TestClient_Delete(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got: %s", r.Method)
		}

		if r.URL.Path != "/api/v1/events/e1" {
			t.Errorf("expected path /api/v1/events/e1, got: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		apiKey:     "test-key",
		athleteID:  "i123",
		httpClient: server.Client(),
	}

	body, err := c.Delete(context.Background(), "/api/v1/events/e1", nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if string(body) != `{}` {
		t.Fatalf("expected response body, got: %s", string(body))
	}
}

func TestClient_ErrorResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"secret internal details"}`))
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		apiKey:     "test-key",
		athleteID:  "i123",
		httpClient: server.Client(),
	}

	_, err := c.Get(context.Background(), "/api/v1/missing", nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	if !strings.Contains(err.Error(), "status 404") {
		t.Fatalf("expected error to contain status 404, got: %v", err)
	}

	if strings.Contains(err.Error(), "secret internal details") {
		t.Fatal("error message should not contain API response body content")
	}
}

func TestClient_BodyCloseError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		apiKey:     "test-key",
		athleteID:  "i123",
		httpClient: server.Client(),
	}

	body, err := c.Get(context.Background(), "/api/v1/test", nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if string(body) != `{"ok":true}` {
		t.Fatalf("expected response body, got: %s", string(body))
	}
}

func TestClient_ResponseExceedsMaxSize(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write more than maxResponseSize (10 MB)
		largeBody := make([]byte, maxResponseSize+1)
		for i := range largeBody {
			largeBody[i] = 'x'
		}
		_, _ = w.Write(largeBody)
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		apiKey:     "test-key",
		athleteID:  "i123",
		httpClient: server.Client(),
	}

	_, err := c.Get(context.Background(), "/api/v1/large", nil)
	if err == nil {
		t.Fatal("expected error for oversized response")
	}

	if !errors.Is(err, errResponseTruncated) {
		t.Fatalf("expected errResponseTruncated, got: %v", err)
	}
}

func TestClient_CanceledContext(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		apiKey:     "test-key",
		athleteID:  "i123",
		httpClient: server.Client(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Get(ctx, "/api/v1/test", nil)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got: %v", err)
	}
}

func TestClient_GetWithNilQueryParams(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query string, got: %s", r.URL.RawQuery)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		apiKey:     "test-key",
		athleteID:  "i123",
		httpClient: server.Client(),
	}

	body, err := c.Get(context.Background(), "/api/v1/test", nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if string(body) != `{"ok":true}` {
		t.Fatalf("expected response body, got: %s", string(body))
	}
}

func TestClient_DeleteWithQueryParams(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got: %s", r.Method)
		}

		if r.URL.Query().Get("force") != "true" {
			t.Errorf("expected force=true query param, got: %s", r.URL.Query().Get("force"))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"deleted":true}`))
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		apiKey:     "test-key",
		athleteID:  "i123",
		httpClient: server.Client(),
	}

	params := url.Values{"force": {"true"}}
	body, err := c.Delete(context.Background(), "/api/v1/events/e1", params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if string(body) != `{"deleted":true}` {
		t.Fatalf("expected response body, got: %s", string(body))
	}
}
