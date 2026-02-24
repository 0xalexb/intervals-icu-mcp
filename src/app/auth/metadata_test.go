package auth

import (
	"encoding/json"
	"testing"
)

func TestNewAuthorizationServerMetadata_EndpointsFromIssuer(t *testing.T) {
	t.Parallel()

	meta := NewAuthorizationServerMetadata("https://auth.example.com")

	if meta.Issuer != "https://auth.example.com" {
		t.Fatalf("Issuer: expected 'https://auth.example.com', got %q", meta.Issuer)
	}

	if meta.AuthorizationEndpoint != "https://auth.example.com/oauth/authorize" {
		t.Fatalf("AuthorizationEndpoint: expected 'https://auth.example.com/oauth/authorize', got %q", meta.AuthorizationEndpoint)
	}

	if meta.TokenEndpoint != "https://auth.example.com/oauth/token" {
		t.Fatalf("TokenEndpoint: expected 'https://auth.example.com/oauth/token', got %q", meta.TokenEndpoint)
	}

	if meta.RegistrationEndpoint != "https://auth.example.com/oauth/register" {
		t.Fatalf("RegistrationEndpoint: expected 'https://auth.example.com/oauth/register', got %q", meta.RegistrationEndpoint)
	}
}

func TestNewAuthorizationServerMetadata_IssuerWithPort(t *testing.T) {
	t.Parallel()

	meta := NewAuthorizationServerMetadata("http://localhost:8080")

	if meta.Issuer != "http://localhost:8080" {
		t.Fatalf("Issuer: expected 'http://localhost:8080', got %q", meta.Issuer)
	}

	if meta.AuthorizationEndpoint != "http://localhost:8080/oauth/authorize" {
		t.Fatalf("AuthorizationEndpoint: expected 'http://localhost:8080/oauth/authorize', got %q", meta.AuthorizationEndpoint)
	}

	if meta.TokenEndpoint != "http://localhost:8080/oauth/token" {
		t.Fatalf("TokenEndpoint: expected 'http://localhost:8080/oauth/token', got %q", meta.TokenEndpoint)
	}

	if meta.RegistrationEndpoint != "http://localhost:8080/oauth/register" {
		t.Fatalf("RegistrationEndpoint: expected 'http://localhost:8080/oauth/register', got %q", meta.RegistrationEndpoint)
	}
}

func TestNewAuthorizationServerMetadata_SupportedValues(t *testing.T) {
	t.Parallel()

	meta := NewAuthorizationServerMetadata("https://auth.example.com")

	assertStringSlice(t, "ResponseTypesSupported", meta.ResponseTypesSupported, []string{"code"})
	assertStringSlice(t, "GrantTypesSupported", meta.GrantTypesSupported, []string{"authorization_code", "refresh_token"})
	assertStringSlice(t, "TokenEndpointAuthMethodsSupported", meta.TokenEndpointAuthMethodsSupported, []string{"none"})
	assertStringSlice(t, "CodeChallengeMethodsSupported", meta.CodeChallengeMethodsSupported, []string{"S256"})
	assertStringSlice(t, "ScopesSupported", meta.ScopesSupported, []string{"mcp"})
}

func TestNewAuthorizationServerMetadata_JSONSerialization(t *testing.T) {
	t.Parallel()

	meta := NewAuthorizationServerMetadata("https://auth.example.com")

	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	expectedKeys := []string{
		"issuer",
		"authorization_endpoint",
		"token_endpoint",
		"registration_endpoint",
		"response_types_supported",
		"grant_types_supported",
		"token_endpoint_auth_methods_supported",
		"code_challenge_methods_supported",
		"scopes_supported",
	}

	for _, key := range expectedKeys {
		if _, ok := parsed[key]; !ok {
			t.Fatalf("expected JSON key %q to be present", key)
		}
	}

	if parsed["issuer"] != "https://auth.example.com" {
		t.Fatalf("JSON issuer: expected 'https://auth.example.com', got %v", parsed["issuer"])
	}

	if parsed["authorization_endpoint"] != "https://auth.example.com/oauth/authorize" {
		t.Fatalf("JSON authorization_endpoint: expected 'https://auth.example.com/oauth/authorize', got %v", parsed["authorization_endpoint"])
	}
}

func TestNewAuthorizationServerMetadata_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := NewAuthorizationServerMetadata("https://auth.example.com")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var roundTripped AuthorizationServerMetadata
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if roundTripped.Issuer != original.Issuer {
		t.Fatalf("Issuer mismatch after round-trip: expected %q, got %q", original.Issuer, roundTripped.Issuer)
	}

	if roundTripped.AuthorizationEndpoint != original.AuthorizationEndpoint {
		t.Fatalf("AuthorizationEndpoint mismatch after round-trip: expected %q, got %q", original.AuthorizationEndpoint, roundTripped.AuthorizationEndpoint)
	}

	if roundTripped.TokenEndpoint != original.TokenEndpoint {
		t.Fatalf("TokenEndpoint mismatch after round-trip: expected %q, got %q", original.TokenEndpoint, roundTripped.TokenEndpoint)
	}

	if roundTripped.RegistrationEndpoint != original.RegistrationEndpoint {
		t.Fatalf("RegistrationEndpoint mismatch after round-trip: expected %q, got %q", original.RegistrationEndpoint, roundTripped.RegistrationEndpoint)
	}
}

func TestNewProtectedResourceMetadata_Fields(t *testing.T) {
	t.Parallel()

	meta := NewProtectedResourceMetadata("https://auth.example.com")

	if meta.Resource != "https://auth.example.com" {
		t.Fatalf("Resource: expected 'https://auth.example.com', got %q", meta.Resource)
	}

	assertStringSlice(t, "AuthorizationServers", meta.AuthorizationServers, []string{"https://auth.example.com"})
	assertStringSlice(t, "ScopesSupported", meta.ScopesSupported, []string{"mcp"})
	assertStringSlice(t, "BearerMethodsSupported", meta.BearerMethodsSupported, []string{"header"})
}

func TestNewProtectedResourceMetadata_IssuerWithPort(t *testing.T) {
	t.Parallel()

	meta := NewProtectedResourceMetadata("http://localhost:8080")

	if meta.Resource != "http://localhost:8080" {
		t.Fatalf("Resource: expected 'http://localhost:8080', got %q", meta.Resource)
	}

	if len(meta.AuthorizationServers) != 1 || meta.AuthorizationServers[0] != "http://localhost:8080" {
		t.Fatalf("AuthorizationServers: expected ['http://localhost:8080'], got %v", meta.AuthorizationServers)
	}
}

func TestNewProtectedResourceMetadata_JSONSerialization(t *testing.T) {
	t.Parallel()

	meta := NewProtectedResourceMetadata("https://auth.example.com")

	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if parsed["resource"] != "https://auth.example.com" {
		t.Fatalf("JSON resource: expected 'https://auth.example.com', got %v", parsed["resource"])
	}

	servers, ok := parsed["authorization_servers"].([]any)
	if !ok || len(servers) != 1 {
		t.Fatalf("JSON authorization_servers: expected array with 1 element, got %v", parsed["authorization_servers"])
	}

	if servers[0] != "https://auth.example.com" {
		t.Fatalf("JSON authorization_servers[0]: expected 'https://auth.example.com', got %v", servers[0])
	}
}

func assertStringSlice(t *testing.T, name string, got, expected []string) {
	t.Helper()

	if len(got) != len(expected) {
		t.Fatalf("%s: expected %d elements, got %d: %v", name, len(expected), len(got), got)
	}

	for i, exp := range expected {
		if got[i] != exp {
			t.Fatalf("%s[%d]: expected %q, got %q", name, i, exp, got[i])
		}
	}
}
