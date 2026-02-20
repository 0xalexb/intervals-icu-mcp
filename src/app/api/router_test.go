package api

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func localhostOrigins() AllowedOrigins {
	return AllowedOrigins{"http://localhost:3000", "http://localhost:8080", "http://127.0.0.1:8080", "http://[::1]:9090"}
}

func TestNewRouter_MCPHandlerReachable(t *testing.T) {
	t.Parallel()

	called := false
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler, localhostOrigins())

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

	router := NewRouter(mcpHandler, localhostOrigins())

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

	router := NewRouter(mcpHandler, localhostOrigins())

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

	router := NewRouter(mcpHandler, localhostOrigins())

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

func TestNewRouter_RecoveryCatchesPanics(t *testing.T) {
	t.Parallel()

	mcpHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	})

	router := NewRouter(mcpHandler, localhostOrigins())

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestNewRouter_RequestIDPresent(t *testing.T) {
	t.Parallel()

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler, localhostOrigins())

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("expected X-Request-ID header to be present on response")
	}
}

func TestNewRouter_CORSPreflightHeaders(t *testing.T) {
	t.Parallel()

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler, localhostOrigins())

	req := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", rec.Code)
	}

	allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:3000" {
		t.Fatalf("expected Access-Control-Allow-Origin 'http://localhost:3000', got %q", allowOrigin)
	}

	allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Fatal("expected Access-Control-Allow-Methods header to be present")
	}

	allowHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	for _, h := range []string{"Mcp-Session-Id", "Last-Event-ID", "Mcp-Protocol-Version"} {
		if !strings.Contains(allowHeaders, h) {
			t.Fatalf("expected Access-Control-Allow-Headers to contain %q, got %q", h, allowHeaders)
		}
	}

	maxAge := rec.Header().Get("Access-Control-Max-Age")
	if maxAge != "86400" {
		t.Fatalf("expected Access-Control-Max-Age '86400', got %q", maxAge)
	}

	vary := rec.Header().Get("Vary")
	if !strings.Contains(vary, "Origin") {
		t.Fatalf("expected Vary header to contain 'Origin', got %q", vary)
	}
}

func TestNewRouter_ExposeHeadersMcpSessionId(t *testing.T) {
	t.Parallel()

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler, localhostOrigins())

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	exposed := rec.Header().Get("Access-Control-Expose-Headers")
	if !strings.Contains(exposed, "Mcp-Session-Id") {
		t.Fatalf("expected Access-Control-Expose-Headers to contain 'Mcp-Session-Id', got %q", exposed)
	}
}

func TestNewRouter_CORSRejectsNonLocalhostOrigin(t *testing.T) {
	t.Parallel()

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler, localhostOrigins())

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Origin", "http://evil.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no Access-Control-Allow-Origin for non-localhost origin, got %q",
			rec.Header().Get("Access-Control-Allow-Origin"))
	}

	if rec.Header().Get("Access-Control-Expose-Headers") != "" {
		t.Fatalf("expected no Access-Control-Expose-Headers for non-localhost origin, got %q",
			rec.Header().Get("Access-Control-Expose-Headers"))
	}
}

func TestNewRouter_CORSAllowsLoopbackVariants(t *testing.T) {
	t.Parallel()

	origins := []string{
		"http://localhost:3000",
		"http://127.0.0.1:8080",
		"http://[::1]:9090",
	}

	for _, origin := range origins {
		mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := NewRouter(mcpHandler, localhostOrigins())

		req := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("origin %q: expected 204 for preflight, got %d", origin, rec.Code)
		}

		allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin != origin {
			t.Fatalf("origin %q: expected Access-Control-Allow-Origin %q, got %q", origin, origin, allowOrigin)
		}
	}
}

func TestNewRouter_CompressGzip(t *testing.T) {
	t.Parallel()

	largeBody := strings.Repeat("hello world ", 100)

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(largeBody))
	})

	router := NewRouter(mcpHandler, localhostOrigins())

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if enc := rec.Header().Get("Content-Encoding"); enc != "gzip" {
		t.Fatalf("expected Content-Encoding 'gzip', got %q", enc)
	}

	gr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	decompressed, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("failed to decompress response: %v", err)
	}

	if string(decompressed) != largeBody {
		t.Fatal("decompressed body does not match original")
	}
}

func TestNewRouter_RateLimitExceeded(t *testing.T) {
	t.Parallel()

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler, localhostOrigins())

	for range 200 {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestNewRouter_MaxRequestSizeRejects(t *testing.T) {
	t.Parallel()

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Request Too Large", http.StatusRequestEntityTooLarge)

			return
		}
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler, localhostOrigins())

	oversized := bytes.NewReader(make([]byte, 1048576+1))
	req := httptest.NewRequest(http.MethodPost, "/mcp", oversized)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestNewRouter_CORSAllowsCustomOrigin(t *testing.T) {
	t.Parallel()

	customOrigins := AllowedOrigins{"https://example.com"}

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler, customOrigins)

	req := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", rec.Code)
	}

	allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://example.com" {
		t.Fatalf("expected Access-Control-Allow-Origin 'https://example.com', got %q", allowOrigin)
	}
}

func TestNewRouter_CORSRejectsUnlistedOrigin(t *testing.T) {
	t.Parallel()

	customOrigins := AllowedOrigins{"https://example.com"}

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(mcpHandler, customOrigins)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Origin", "http://evil.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no Access-Control-Allow-Origin for unlisted origin, got %q",
			rec.Header().Get("Access-Control-Allow-Origin"))
	}

	if rec.Header().Get("Access-Control-Expose-Headers") != "" {
		t.Fatalf("expected no Access-Control-Expose-Headers for unlisted origin, got %q",
			rec.Header().Get("Access-Control-Expose-Headers"))
	}
}
