package auth

import (
	"testing"
	"time"
)

func TestSaveAuthCode_And_ConsumeAuthCode(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	code := &Code{
		Code:                "test-code",
		ClientID:            "client-1",
		RedirectURI:         "http://localhost/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		GitHubUsername:       "alice",
		Scopes:              []string{"mcp"},
		ExpiresAt:           now.Add(10 * time.Minute),
	}

	store.SaveAuthCode(code)

	got, err := store.ConsumeAuthCode("test-code", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Code != "test-code" {
		t.Fatalf("expected code 'test-code', got %q", got.Code)
	}

	if got.ClientID != "client-1" {
		t.Fatalf("expected client ID 'client-1', got %q", got.ClientID)
	}

	if got.GitHubUsername != "alice" {
		t.Fatalf("expected GitHub username 'alice', got %q", got.GitHubUsername)
	}
}

func TestConsumeAuthCode_OneTimeUse(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	code := &Code{
		Code:      "one-time-code",
		ClientID:  "client-1",
		ExpiresAt: now.Add(10 * time.Minute),
	}

	store.SaveAuthCode(code)

	_, err := store.ConsumeAuthCode("one-time-code", now)
	if err != nil {
		t.Fatalf("first consume should succeed: %v", err)
	}

	_, err = store.ConsumeAuthCode("one-time-code", now)
	if err == nil {
		t.Fatal("second consume should fail")
	}

	if err != errAuthCodeNotFound {
		t.Fatalf("expected errAuthCodeNotFound, got %v", err)
	}
}

func TestConsumeAuthCode_Expired(t *testing.T) {
	t.Parallel()

	store := NewStore()
	past := time.Now().Add(-10 * time.Minute)

	code := &Code{
		Code:      "expired-code",
		ClientID:  "client-1",
		ExpiresAt: past,
	}

	store.SaveAuthCode(code)

	_, err := store.ConsumeAuthCode("expired-code", time.Now())
	if err == nil {
		t.Fatal("expected error for expired code")
	}

	if err != errAuthCodeExpired {
		t.Fatalf("expected errAuthCodeExpired, got %v", err)
	}
}

func TestConsumeAuthCode_ExpiredCodeIsDeletedFromStore(t *testing.T) {
	t.Parallel()

	store := NewStore()
	past := time.Now().Add(-10 * time.Minute)

	code := &Code{
		Code:      "expired-deleted-code",
		ClientID:  "client-1",
		ExpiresAt: past,
	}

	store.SaveAuthCode(code)

	_, _ = store.ConsumeAuthCode("expired-deleted-code", time.Now())

	_, err := store.ConsumeAuthCode("expired-deleted-code", time.Now())
	if err != errAuthCodeNotFound {
		t.Fatalf("expected errAuthCodeNotFound after consuming expired code, got %v", err)
	}
}

func TestConsumeAuthCode_NotFound(t *testing.T) {
	t.Parallel()

	store := NewStore()

	_, err := store.ConsumeAuthCode("nonexistent", time.Now())
	if err == nil {
		t.Fatal("expected error for nonexistent code")
	}

	if err != errAuthCodeNotFound {
		t.Fatalf("expected errAuthCodeNotFound, got %v", err)
	}
}

func TestSaveRefreshToken_And_ConsumeRefreshToken(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	token := &RefreshToken{
		Token:         "refresh-token-1",
		ClientID:      "client-1",
		GitHubUsername: "bob",
		Scopes:        []string{"mcp"},
		ExpiresAt:     now.Add(24 * time.Hour),
	}

	store.SaveRefreshToken(token)

	got, err := store.ConsumeRefreshToken("refresh-token-1", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Token != "refresh-token-1" {
		t.Fatalf("expected token 'refresh-token-1', got %q", got.Token)
	}

	if got.GitHubUsername != "bob" {
		t.Fatalf("expected GitHub username 'bob', got %q", got.GitHubUsername)
	}
}

func TestConsumeRefreshToken_Rotation(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	token := &RefreshToken{
		Token:     "old-refresh-token",
		ClientID:  "client-1",
		ExpiresAt: now.Add(24 * time.Hour),
	}

	store.SaveRefreshToken(token)

	_, err := store.ConsumeRefreshToken("old-refresh-token", now)
	if err != nil {
		t.Fatalf("first consume should succeed: %v", err)
	}

	_, err = store.ConsumeRefreshToken("old-refresh-token", now)
	if err == nil {
		t.Fatal("second consume should fail after rotation")
	}

	if err != errRefreshTokenNotFound {
		t.Fatalf("expected errRefreshTokenNotFound, got %v", err)
	}

	newToken := &RefreshToken{
		Token:     "new-refresh-token",
		ClientID:  "client-1",
		ExpiresAt: now.Add(24 * time.Hour),
	}

	store.SaveRefreshToken(newToken)

	got, err := store.ConsumeRefreshToken("new-refresh-token", now)
	if err != nil {
		t.Fatalf("consuming new token should succeed: %v", err)
	}

	if got.Token != "new-refresh-token" {
		t.Fatalf("expected 'new-refresh-token', got %q", got.Token)
	}
}

func TestConsumeRefreshToken_Expired(t *testing.T) {
	t.Parallel()

	store := NewStore()
	past := time.Now().Add(-24 * time.Hour)

	token := &RefreshToken{
		Token:     "expired-refresh",
		ClientID:  "client-1",
		ExpiresAt: past,
	}

	store.SaveRefreshToken(token)

	_, err := store.ConsumeRefreshToken("expired-refresh", time.Now())
	if err == nil {
		t.Fatal("expected error for expired refresh token")
	}

	if err != errRefreshTokenExpired {
		t.Fatalf("expected errRefreshTokenExpired, got %v", err)
	}
}

func TestConsumeRefreshToken_ExpiredTokenIsDeletedFromStore(t *testing.T) {
	t.Parallel()

	store := NewStore()
	past := time.Now().Add(-24 * time.Hour)

	token := &RefreshToken{
		Token:     "expired-deleted-refresh",
		ClientID:  "client-1",
		ExpiresAt: past,
	}

	store.SaveRefreshToken(token)

	_, _ = store.ConsumeRefreshToken("expired-deleted-refresh", time.Now())

	_, err := store.ConsumeRefreshToken("expired-deleted-refresh", time.Now())
	if err != errRefreshTokenNotFound {
		t.Fatalf("expected errRefreshTokenNotFound after consuming expired token, got %v", err)
	}
}

func TestConsumeRefreshToken_NotFound(t *testing.T) {
	t.Parallel()

	store := NewStore()

	_, err := store.ConsumeRefreshToken("nonexistent", time.Now())
	if err == nil {
		t.Fatal("expected error for nonexistent token")
	}

	if err != errRefreshTokenNotFound {
		t.Fatalf("expected errRefreshTokenNotFound, got %v", err)
	}
}

func TestSaveClient_And_GetClient(t *testing.T) {
	t.Parallel()

	store := NewStore()

	client := &RegisteredClient{
		ClientID:     "client-abc",
		RedirectURIs: []string{"http://localhost/callback"},
		ClientName:   "Test App",
		GrantTypes:   []string{"authorization_code", "refresh_token"},
		CreatedAt:    time.Now(),
	}

	store.SaveClient(client)

	got, err := store.GetClient("client-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ClientID != "client-abc" {
		t.Fatalf("expected client ID 'client-abc', got %q", got.ClientID)
	}

	if got.ClientName != "Test App" {
		t.Fatalf("expected client name 'Test App', got %q", got.ClientName)
	}

	if len(got.RedirectURIs) != 1 || got.RedirectURIs[0] != "http://localhost/callback" {
		t.Fatalf("expected redirect URIs [http://localhost/callback], got %v", got.RedirectURIs)
	}

	if len(got.GrantTypes) != 2 {
		t.Fatalf("expected 2 grant types, got %d", len(got.GrantTypes))
	}
}

func TestGetClient_NotFound(t *testing.T) {
	t.Parallel()

	store := NewStore()

	_, err := store.GetClient("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent client")
	}

	if err != errClientNotFound {
		t.Fatalf("expected errClientNotFound, got %v", err)
	}
}

func TestSaveClient_Overwrite(t *testing.T) {
	t.Parallel()

	store := NewStore()

	client1 := &RegisteredClient{
		ClientID:   "client-overwrite",
		ClientName: "Original",
		CreatedAt:  time.Now(),
	}

	store.SaveClient(client1)

	client2 := &RegisteredClient{
		ClientID:   "client-overwrite",
		ClientName: "Updated",
		CreatedAt:  time.Now(),
	}

	store.SaveClient(client2)

	got, err := store.GetClient("client-overwrite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ClientName != "Updated" {
		t.Fatalf("expected client name 'Updated', got %q", got.ClientName)
	}
}

func TestNewStore_EmptyMaps(t *testing.T) {
	t.Parallel()

	store := NewStore()

	_, err := store.ConsumeAuthCode("any", time.Now())
	if err != errAuthCodeNotFound {
		t.Fatalf("expected errAuthCodeNotFound on empty store, got %v", err)
	}

	_, err = store.ConsumeRefreshToken("any", time.Now())
	if err != errRefreshTokenNotFound {
		t.Fatalf("expected errRefreshTokenNotFound on empty store, got %v", err)
	}

	_, err = store.GetClient("any")
	if err != errClientNotFound {
		t.Fatalf("expected errClientNotFound on empty store, got %v", err)
	}
}
