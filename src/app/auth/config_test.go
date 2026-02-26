package auth

import (
	"errors"
	"testing"
)

func TestNewAllowedUsers_ValidSingleUser(t *testing.T) {
	t.Parallel()

	users := NewAllowedUsers("alice")

	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	if users[0] != "alice" {
		t.Fatalf("expected 'alice', got %q", users[0])
	}
}

func TestNewAllowedUsers_ValidMultipleUsers(t *testing.T) {
	t.Parallel()

	users := NewAllowedUsers("alice,bob,charlie")

	expected := []string{"alice", "bob", "charlie"}
	if len(users) != len(expected) {
		t.Fatalf("expected %d users, got %d", len(expected), len(users))
	}

	for i, exp := range expected {
		if users[i] != exp {
			t.Fatalf("user[%d]: expected %q, got %q", i, exp, users[i])
		}
	}
}

func TestNewAllowedUsers_LowercasesEntries(t *testing.T) {
	t.Parallel()

	users := NewAllowedUsers("Alice,BOB,Charlie")

	expected := []string{"alice", "bob", "charlie"}
	if len(users) != len(expected) {
		t.Fatalf("expected %d users, got %d", len(expected), len(users))
	}

	for i, exp := range expected {
		if users[i] != exp {
			t.Fatalf("user[%d]: expected %q, got %q", i, exp, users[i])
		}
	}
}

func TestNewAllowedUsers_TrimsWhitespace(t *testing.T) {
	t.Parallel()

	users := NewAllowedUsers("  alice , bob ,  charlie  ")

	expected := []string{"alice", "bob", "charlie"}
	if len(users) != len(expected) {
		t.Fatalf("expected %d users, got %d", len(expected), len(users))
	}

	for i, exp := range expected {
		if users[i] != exp {
			t.Fatalf("user[%d]: expected %q, got %q", i, exp, users[i])
		}
	}
}

func TestNewAllowedUsers_DropsEmptyEntries(t *testing.T) {
	t.Parallel()

	users := NewAllowedUsers("alice,,bob,")

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	if users[0] != "alice" || users[1] != "bob" {
		t.Fatalf("expected [alice, bob], got %v", users)
	}
}

func TestNewAllowedUsers_EmptyStringProducesEmptyList(t *testing.T) {
	t.Parallel()

	users := NewAllowedUsers("")

	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}
}

func TestNewAllowedUsers_WhitespaceOnlyProducesEmptyList(t *testing.T) {
	t.Parallel()

	users := NewAllowedUsers("   ")

	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}
}

func TestAllowedUsers_Contains_ExactMatch(t *testing.T) {
	t.Parallel()

	users := NewAllowedUsers("alice,bob")

	if !users.Contains("alice") {
		t.Fatal("expected Contains('alice') to be true")
	}

	if !users.Contains("bob") {
		t.Fatal("expected Contains('bob') to be true")
	}
}

func TestAllowedUsers_Contains_CaseInsensitive(t *testing.T) {
	t.Parallel()

	users := NewAllowedUsers("alice,bob")

	if !users.Contains("Alice") {
		t.Fatal("expected Contains('Alice') to be true")
	}

	if !users.Contains("BOB") {
		t.Fatal("expected Contains('BOB') to be true")
	}

	if !users.Contains("ALICE") {
		t.Fatal("expected Contains('ALICE') to be true")
	}
}

func TestAllowedUsers_Contains_NotFound(t *testing.T) {
	t.Parallel()

	users := NewAllowedUsers("alice,bob")

	if users.Contains("charlie") {
		t.Fatal("expected Contains('charlie') to be false")
	}
}

func TestAllowedUsers_Contains_EmptyList(t *testing.T) {
	t.Parallel()

	users := NewAllowedUsers("")

	if users.Contains("alice") {
		t.Fatal("expected Contains('alice') on empty list to be false")
	}
}

func TestNewValidatedIssuer_ValidHTTPS(t *testing.T) {
	t.Parallel()

	issuer, err := NewValidatedIssuer("https://auth.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if issuer != "https://auth.example.com" {
		t.Fatalf("expected 'https://auth.example.com', got %q", issuer)
	}
}

func TestNewValidatedIssuer_ValidHTTP(t *testing.T) {
	t.Parallel()

	issuer, err := NewValidatedIssuer("http://localhost:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if issuer != "http://localhost:8080" {
		t.Fatalf("expected 'http://localhost:8080', got %q", issuer)
	}
}

func TestNewValidatedIssuer_MissingScheme(t *testing.T) {
	t.Parallel()

	_, err := NewValidatedIssuer("example.com")
	if err == nil {
		t.Fatal("expected error for missing scheme, got nil")
	}
}

func TestNewValidatedIssuer_InvalidScheme(t *testing.T) {
	t.Parallel()

	_, err := NewValidatedIssuer("ftp://example.com")
	if err == nil {
		t.Fatal("expected error for ftp scheme, got nil")
	}
}

func TestNewValidatedIssuer_EmptyHost(t *testing.T) {
	t.Parallel()

	_, err := NewValidatedIssuer("http://")
	if err == nil {
		t.Fatal("expected error for empty host, got nil")
	}
}

func TestNewValidatedIssuer_EmptyString(t *testing.T) {
	t.Parallel()

	_, err := NewValidatedIssuer("")
	if err == nil {
		t.Fatal("expected error for empty string, got nil")
	}
}

func TestNewValidatedIssuer_TrailingSlashStripped(t *testing.T) {
	t.Parallel()

	issuer, err := NewValidatedIssuer("http://localhost:8080/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if issuer != "http://localhost:8080" {
		t.Fatalf("expected trailing slash stripped, got %q", issuer)
	}
}

func TestNewValidatedIssuer_RejectsPath(t *testing.T) {
	t.Parallel()

	_, err := NewValidatedIssuer("https://example.com/path")
	if !errors.Is(err, errIssuerHasPath) {
		t.Fatalf("expected errIssuerHasPath, got %v", err)
	}
}

func TestNewValidatedIssuer_RejectsQuery(t *testing.T) {
	t.Parallel()

	_, err := NewValidatedIssuer("https://example.com?foo=bar")
	if !errors.Is(err, errIssuerHasQuery) {
		t.Fatalf("expected errIssuerHasQuery, got %v", err)
	}
}

func TestNewValidatedIssuer_RejectsFragment(t *testing.T) {
	t.Parallel()

	_, err := NewValidatedIssuer("https://example.com#frag")
	if !errors.Is(err, errIssuerHasFragment) {
		t.Fatalf("expected errIssuerHasFragment, got %v", err)
	}
}

func TestNewJWTSecret_TooShort(t *testing.T) {
	t.Parallel()

	_, err := NewJWTSecret("short")
	if !errors.Is(err, errJWTSecretTooShort) {
		t.Fatalf("expected errJWTSecretTooShort, got %v", err)
	}
}

func TestNewJWTSecret_ExplicitValue(t *testing.T) {
	t.Parallel()

	secret, err := NewJWTSecret("my-secret-key-that-is-at-least-32-chars-long")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if secret != "my-secret-key-that-is-at-least-32-chars-long" {
		t.Fatalf("expected explicit secret value, got %q", secret)
	}
}

func TestNewJWTSecret_AutoGeneratesWhenEmpty(t *testing.T) {
	t.Parallel()

	secret, err := NewJWTSecret("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if secret == "" {
		t.Fatal("expected non-empty auto-generated secret")
	}

	if len(secret) < 32 {
		t.Fatalf("expected auto-generated secret to be at least 32 chars, got %d", len(secret))
	}
}

func TestNewJWTSecret_AutoGeneratesWhenWhitespace(t *testing.T) {
	t.Parallel()

	secret, err := NewJWTSecret("   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if secret == "" || secret == "   " {
		t.Fatal("expected auto-generated secret, not empty or whitespace")
	}
}

func TestNewJWTSecret_AutoGeneratedSecretsAreUnique(t *testing.T) {
	t.Parallel()

	secret1, err := NewJWTSecret("")
	if err != nil {
		t.Fatalf("unexpected error generating first secret: %v", err)
	}

	secret2, err := NewJWTSecret("")
	if err != nil {
		t.Fatalf("unexpected error generating second secret: %v", err)
	}

	if secret1 == secret2 {
		t.Fatal("expected two auto-generated secrets to be different")
	}
}
