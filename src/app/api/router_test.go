package api

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xalexb/intervals-icu-mcp/src/app/api/rest"
	appauth "github.com/0xalexb/intervals-icu-mcp/src/app/auth"
	ghclient "github.com/0xalexb/intervals-icu-mcp/src/app/clients/github"
)

const testIssuer = appauth.Issuer("http://localhost:8080")

func localhostOrigins() AllowedOrigins {
	return AllowedOrigins{"http://localhost:3000", "http://localhost:8080", "http://127.0.0.1:8080", "http://[::1]:9090"}
}

func testRouterParams(mcpHandler http.Handler, origins AllowedOrigins) RouterParams {
	secret := appauth.JWTSecret("test-secret-for-router-tests")
	store := appauth.NewStore()
	metadata := appauth.NewAuthorizationServerMetadata(testIssuer)
	prMetadata := appauth.NewProtectedResourceMetadata(testIssuer)
	verifier := appauth.NewTokenVerifier(secret, testIssuer)

	handler, err := rest.NewHandler(rest.HandlerParams{
		Store:                       store,
		AllowedUsers:                appauth.AllowedUsers{},
		GitHubClientID:              "test-client-id",
		GitHubClientSecret:          "test-client-secret",
		JWTSecret:                   secret,
		Issuer:                      testIssuer,
		AuthorizationServerMetadata: metadata,
		GitHubClient:                ghclient.NewClient(),
	})
	if err != nil {
		panic("creating test handler: " + err.Error())
	}

	return RouterParams{
		MCPHandler:                  mcpHandler,
		Origins:                     origins,
		AuthHandler:                 handler,
		AuthorizationServerMetadata: metadata,
		ProtectedResourceMetadata:   prMetadata,
		TokenVerifier:               verifier,
		Issuer:                      testIssuer,
	}
}

func TestNewRouter_MCPHandlerRequiresAuth(t *testing.T) {
	t.Parallel()

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth token, got %d", rec.Code)
	}
}

func TestNewRouter_MCPHandlerReachableWithValidToken(t *testing.T) {
	t.Parallel()

	called := false
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	secret := appauth.JWTSecret("test-secret-for-router-tests")
	token, err := appauth.IssueAccessToken(secret, testIssuer, 3600_000_000_000, "testuser", []string{"mcp"})
	if err != nil {
		t.Fatalf("failed to issue test token: %v", err)
	}

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected MCP handler to be called for /mcp with valid token")
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestNewRouter_MCPSubpathRequiresAuth(t *testing.T) {
	t.Parallel()

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

	req := httptest.NewRequest(http.MethodGet, "/mcp/sse", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth token for /mcp/sse, got %d", rec.Code)
	}
}

func TestNewRouter_OtherPathNotReachable(t *testing.T) {
	t.Parallel()

	called := false
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

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

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

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

	secret := appauth.JWTSecret("test-secret-for-router-tests")
	token, err := appauth.IssueAccessToken(secret, testIssuer, 3600_000_000_000, "testuser", []string{"mcp"})
	if err != nil {
		t.Fatalf("issuing access token: %v", err)
	}

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
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

	secret := appauth.JWTSecret("test-secret-for-router-tests")
	token, err := appauth.IssueAccessToken(secret, testIssuer, 3600_000_000_000, "testuser", []string{"mcp"})
	if err != nil {
		t.Fatalf("issuing access token: %v", err)
	}

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
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

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

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

	secret := appauth.JWTSecret("test-secret-for-router-tests")
	token, err := appauth.IssueAccessToken(secret, testIssuer, 3600_000_000_000, "testuser", []string{"mcp"})
	if err != nil {
		t.Fatalf("issuing access token: %v", err)
	}

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	req.Header.Set("Authorization", "Bearer "+token)
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

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

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

		router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

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

	secret := appauth.JWTSecret("test-secret-for-router-tests")
	token, err := appauth.IssueAccessToken(secret, testIssuer, 3600_000_000_000, "testuser", []string{"mcp"})
	if err != nil {
		t.Fatalf("issuing access token: %v", err)
	}

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Authorization", "Bearer "+token)
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

	secret := appauth.JWTSecret("test-secret-for-router-tests")
	token, err := appauth.IssueAccessToken(secret, testIssuer, 3600_000_000_000, "testuser", []string{"mcp"})
	if err != nil {
		t.Fatalf("issuing access token: %v", err)
	}

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

	// Per-IP sliding window: effective limit = rateLimitRate + rateLimitBurst = 100 + 200 = 300.
	for range 300 {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestNewRouter_RegisterRateLimitExceeded(t *testing.T) {
	t.Parallel()

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

	// Per-IP sliding window: effective limit = registerRateLimitRate + registerRateLimitBurst = 2 + 5 = 7.
	for range 7 {
		req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(`{"redirect_uris":["http://localhost/cb"]}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	}

	// The next request should be rate-limited.
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(`{"redirect_uris":["http://localhost/cb"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for register after per-IP limit exhaustion, got %d", rec.Code)
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

	secret := appauth.JWTSecret("test-secret-for-router-tests")
	token, err := appauth.IssueAccessToken(secret, testIssuer, 3600_000_000_000, "testuser", []string{"mcp"})
	if err != nil {
		t.Fatalf("issuing access token: %v", err)
	}

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

	oversized := bytes.NewReader(make([]byte, 1048576+1))
	req := httptest.NewRequest(http.MethodPost, "/mcp", oversized)
	req.Header.Set("Authorization", "Bearer "+token)
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

	router := NewRouter(testRouterParams(mcpHandler, customOrigins))

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

func TestNewRouter_CORSRejectsDifferentPortSameHostname(t *testing.T) {
	t.Parallel()

	// Only allow http://localhost:3000 — port 9999 must be rejected.
	origins := AllowedOrigins{"http://localhost:3000"}

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(testRouterParams(mcpHandler, origins))

	req := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
	req.Header.Set("Origin", "http://localhost:9999")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no Access-Control-Allow-Origin for different port on same hostname, got %q",
			rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestNewRouter_CORSRejectsUnlistedOrigin(t *testing.T) {
	t.Parallel()

	customOrigins := AllowedOrigins{"https://example.com"}

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(testRouterParams(mcpHandler, customOrigins))

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

func TestNewRouter_OAuthEndpointsReachable(t *testing.T) {
	t.Parallel()

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/.well-known/oauth-authorization-server"},
		{http.MethodGet, "/.well-known/oauth-protected-resource"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s: expected 200, got %d", ep.method, ep.path, rec.Code)
		}
	}
}

func TestNewRouter_ProtectedResourceMetadataContent(t *testing.T) {
	t.Parallel()

	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(testRouterParams(mcpHandler, localhostOrigins()))

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "http://localhost:8080") {
		t.Fatalf("expected protected resource metadata to contain issuer, got %s", body)
	}
}

