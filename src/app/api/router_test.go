package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouter_MCPHandlerReachable(t *testing.T) {
	t.Parallel()

	called := false
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected MCP handler to be called for /mcp")
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestNewRouter_MCPSubpathReachable(t *testing.T) {
	t.Parallel()

	called := false
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler)

	req := httptest.NewRequest(http.MethodGet, "/mcp/sse", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected MCP handler to be called for /mcp/sse")
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestNewRouter_OtherPathNotReachable(t *testing.T) {
	t.Parallel()

	called := false
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler)

	req := httptest.NewRequest(http.MethodGet, "/other", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if called {
		t.Fatal("expected MCP handler NOT to be called for /other")
	}

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for /other, got %d", rec.Code)
	}
}

func TestNewRouter_RootNotReachable(t *testing.T) {
	t.Parallel()

	called := false
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if called {
		t.Fatal("expected MCP handler NOT to be called for /")
	}

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for /, got %d", rec.Code)
	}
}
