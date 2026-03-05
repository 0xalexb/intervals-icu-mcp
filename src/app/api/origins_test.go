package api

import (
	"testing"
)

func TestNewAllowedOrigins_ValidHTTPWithPort(t *testing.T) {
	t.Parallel()

	origins, err := NewAllowedOrigins("http://localhost:3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(origins) != 1 {
		t.Fatalf("expected 1 origin, got %d", len(origins))
	}

	if origins[0] != "http://localhost:3000" {
		t.Fatalf("expected 'http://localhost:3000', got %q", origins[0])
	}
}

func TestNewAllowedOrigins_ValidHTTPS(t *testing.T) {
	t.Parallel()

	origins, err := NewAllowedOrigins("https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(origins) != 1 {
		t.Fatalf("expected 1 origin, got %d", len(origins))
	}

	if origins[0] != "https://example.com" {
		t.Fatalf("expected 'https://example.com', got %q", origins[0])
	}
}

func TestNewAllowedOrigins_ValidIPWithPort(t *testing.T) {
	t.Parallel()

	origins, err := NewAllowedOrigins("http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(origins) != 1 {
		t.Fatalf("expected 1 origin, got %d", len(origins))
	}

	if origins[0] != "http://127.0.0.1:8080" {
		t.Fatalf("expected 'http://127.0.0.1:8080', got %q", origins[0])
	}
}

func TestNewAllowedOrigins_ValidIPv6WithPort(t *testing.T) {
	t.Parallel()

	origins, err := NewAllowedOrigins("http://[::1]:9090")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(origins) != 1 {
		t.Fatalf("expected 1 origin, got %d", len(origins))
	}

	if origins[0] != "http://[::1]:9090" {
		t.Fatalf("expected 'http://[::1]:9090', got %q", origins[0])
	}
}

func TestNewAllowedOrigins_ValidMultipleOrigins(t *testing.T) {
	t.Parallel()

	origins, err := NewAllowedOrigins("http://localhost:3000,https://example.com,http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"http://localhost:3000", "https://example.com", "http://127.0.0.1:8080"}
	if len(origins) != len(expected) {
		t.Fatalf("expected %d origins, got %d", len(expected), len(origins))
	}

	for i, exp := range expected {
		if origins[i] != exp {
			t.Fatalf("origin[%d]: expected %q, got %q", i, exp, origins[i])
		}
	}
}

func TestNewAllowedOrigins_WhitespaceTrimming(t *testing.T) {
	t.Parallel()

	origins, err := NewAllowedOrigins("  http://localhost:3000 , https://example.com ,  http://127.0.0.1:8080  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"http://localhost:3000", "https://example.com", "http://127.0.0.1:8080"}
	if len(origins) != len(expected) {
		t.Fatalf("expected %d origins, got %d", len(expected), len(origins))
	}

	for i, exp := range expected {
		if origins[i] != exp {
			t.Fatalf("origin[%d]: expected %q, got %q", i, exp, origins[i])
		}
	}
}

func TestNewAllowedOrigins_EmptyStringProducesEmptyList(t *testing.T) {
	t.Parallel()

	origins, err := NewAllowedOrigins("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(origins) != 0 {
		t.Fatalf("expected 0 origins, got %d", len(origins))
	}
}

func TestNewAllowedOrigins_WhitespaceOnlyProducesEmptyList(t *testing.T) {
	t.Parallel()

	origins, err := NewAllowedOrigins("   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(origins) != 0 {
		t.Fatalf("expected 0 origins, got %d", len(origins))
	}
}

func TestNewAllowedOrigins_DropEmptyEntries(t *testing.T) {
	t.Parallel()

	origins, err := NewAllowedOrigins("http://localhost:3000,,https://example.com,")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(origins))
	}

	if origins[0] != "http://localhost:3000" || origins[1] != "https://example.com" {
		t.Fatalf("expected [http://localhost:3000, https://example.com], got %v", origins)
	}
}

func TestNewAllowedOrigins_InvalidBareHostname(t *testing.T) {
	t.Parallel()

	_, err := NewAllowedOrigins("localhost")
	if err == nil {
		t.Fatal("expected error for bare hostname, got nil")
	}
}

func TestNewAllowedOrigins_InvalidMissingScheme(t *testing.T) {
	t.Parallel()

	_, err := NewAllowedOrigins("example.com:3000")
	if err == nil {
		t.Fatal("expected error for origin missing scheme, got nil")
	}
}

func TestNewAllowedOrigins_InvalidWithPath(t *testing.T) {
	t.Parallel()

	_, err := NewAllowedOrigins("http://example.com/path")
	if err == nil {
		t.Fatal("expected error for origin with path, got nil")
	}
}

func TestNewAllowedOrigins_InvalidWithQuery(t *testing.T) {
	t.Parallel()

	_, err := NewAllowedOrigins("http://example.com?q=1")
	if err == nil {
		t.Fatal("expected error for origin with query, got nil")
	}
}

func TestNewAllowedOrigins_InvalidWildcard(t *testing.T) {
	t.Parallel()

	_, err := NewAllowedOrigins("*")
	if err == nil {
		t.Fatal("expected error for wildcard origin, got nil")
	}
}

func TestNewAllowedOrigins_InvalidAmongValid(t *testing.T) {
	t.Parallel()

	_, err := NewAllowedOrigins("http://localhost:3000,evil.com,https://example.com")
	if err == nil {
		t.Fatal("expected error when one origin is invalid, got nil")
	}
}

func TestNewAllowedOrigins_InvalidTrailingSlash(t *testing.T) {
	t.Parallel()

	_, err := NewAllowedOrigins("http://localhost:3000/")
	if err == nil {
		t.Fatal("expected error for origin with trailing slash, got nil")
	}
}

func TestNewAllowedOrigins_InvalidWithFragment(t *testing.T) {
	t.Parallel()

	_, err := NewAllowedOrigins("http://example.com#section")
	if err == nil {
		t.Fatal("expected error for origin with fragment, got nil")
	}
}

