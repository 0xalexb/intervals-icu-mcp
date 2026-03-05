package auth

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

var noValidation = func(*Code) error { return nil }

func TestSaveAuthCode_And_ValidateAndConsumeAuthCode(t *testing.T) {
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

	if err := store.SaveAuthCode(code); err != nil {
		t.Fatalf("saving auth code: %v", err)
	}

	got, err := store.ValidateAndConsumeAuthCode("test-code", now, noValidation)
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

func TestValidateAndConsumeAuthCode_OneTimeUse(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	code := &Code{
		Code:      "one-time-code",
		ClientID:  "client-1",
		ExpiresAt: now.Add(10 * time.Minute),
	}

	if err := store.SaveAuthCode(code); err != nil {
		t.Fatalf("saving auth code: %v", err)
	}

	_, err := store.ValidateAndConsumeAuthCode("one-time-code", now, noValidation)
	if err != nil {
		t.Fatalf("first consume should succeed: %v", err)
	}

	_, err = store.ValidateAndConsumeAuthCode("one-time-code", now, noValidation)
	if err == nil {
		t.Fatal("second consume should fail")
	}

	if err != errAuthCodeNotFound {
		t.Fatalf("expected errAuthCodeNotFound, got %v", err)
	}
}

func TestValidateAndConsumeAuthCode_Expired(t *testing.T) {
	t.Parallel()

	store := NewStore()
	past := time.Now().Add(-10 * time.Minute)

	code := &Code{
		Code:      "expired-code",
		ClientID:  "client-1",
		ExpiresAt: past,
	}

	if err := store.SaveAuthCode(code); err != nil {
		t.Fatalf("saving auth code: %v", err)
	}

	_, err := store.ValidateAndConsumeAuthCode("expired-code", time.Now(), noValidation)
	if err == nil {
		t.Fatal("expected error for expired code")
	}

	if err != errAuthCodeExpired {
		t.Fatalf("expected errAuthCodeExpired, got %v", err)
	}
}

func TestValidateAndConsumeAuthCode_ExpiredCodeIsDeletedFromStore(t *testing.T) {
	t.Parallel()

	store := NewStore()
	past := time.Now().Add(-10 * time.Minute)

	code := &Code{
		Code:      "expired-deleted-code",
		ClientID:  "client-1",
		ExpiresAt: past,
	}

	if err := store.SaveAuthCode(code); err != nil {
		t.Fatalf("saving auth code: %v", err)
	}

	_, _ = store.ValidateAndConsumeAuthCode("expired-deleted-code", time.Now(), noValidation)

	_, err := store.ValidateAndConsumeAuthCode("expired-deleted-code", time.Now(), noValidation)
	if err != errAuthCodeNotFound {
		t.Fatalf("expected errAuthCodeNotFound after consuming expired code, got %v", err)
	}
}

func TestValidateAndConsumeAuthCode_NotFound(t *testing.T) {
	t.Parallel()

	store := NewStore()

	_, err := store.ValidateAndConsumeAuthCode("nonexistent", time.Now(), noValidation)
	if err == nil {
		t.Fatal("expected error for nonexistent code")
	}

	if err != errAuthCodeNotFound {
		t.Fatalf("expected errAuthCodeNotFound, got %v", err)
	}
}

func TestValidateAndConsumeAuthCode_ValidationFailureDoesNotConsume(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	if err := store.SaveAuthCode(&Code{
		Code:      "validate-fail-code",
		ClientID:  "client-1",
		ExpiresAt: now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("saving auth code: %v", err)
	}

	validationErr := errors.New("validation failed")

	_, err := store.ValidateAndConsumeAuthCode("validate-fail-code", now, func(*Code) error {
		return validationErr
	})
	if err != validationErr {
		t.Fatalf("expected validation error, got %v", err)
	}

	// Code should still exist after failed validation.
	got, err := store.GetAuthCode("validate-fail-code", now)
	if err != nil {
		t.Fatalf("code should still exist after validation failure: %v", err)
	}

	if got.Code != "validate-fail-code" {
		t.Fatalf("expected 'validate-fail-code', got %q", got.Code)
	}
}

func TestGetAuthCode_Success(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	code := &Code{
		Code:                "get-code",
		ClientID:            "client-1",
		RedirectURI:         "http://localhost/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		GitHubUsername:       "alice",
		Scopes:              []string{"mcp"},
		ExpiresAt:           now.Add(10 * time.Minute),
	}

	if err := store.SaveAuthCode(code); err != nil {
		t.Fatalf("saving auth code: %v", err)
	}

	got, err := store.GetAuthCode("get-code", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Code != "get-code" {
		t.Fatalf("expected code 'get-code', got %q", got.Code)
	}

	if got.ClientID != "client-1" {
		t.Fatalf("expected client ID 'client-1', got %q", got.ClientID)
	}
}

func TestGetAuthCode_DoesNotDelete(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	if err := store.SaveAuthCode(&Code{
		Code:      "persistent-code",
		ClientID:  "client-1",
		ExpiresAt: now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("saving auth code: %v", err)
	}

	_, err := store.GetAuthCode("persistent-code", now)
	if err != nil {
		t.Fatalf("first get should succeed: %v", err)
	}

	_, err = store.GetAuthCode("persistent-code", now)
	if err != nil {
		t.Fatalf("second get should also succeed (not deleted): %v", err)
	}
}

func TestGetAuthCode_NotFound(t *testing.T) {
	t.Parallel()

	store := NewStore()

	_, err := store.GetAuthCode("nonexistent", time.Now())
	if err != errAuthCodeNotFound {
		t.Fatalf("expected errAuthCodeNotFound, got %v", err)
	}
}

func TestGetAuthCode_Expired(t *testing.T) {
	t.Parallel()

	store := NewStore()
	past := time.Now().Add(-10 * time.Minute)

	if err := store.SaveAuthCode(&Code{
		Code:      "expired-get-code",
		ClientID:  "client-1",
		ExpiresAt: past,
	}); err != nil {
		t.Fatalf("saving auth code: %v", err)
	}

	_, err := store.GetAuthCode("expired-get-code", time.Now())
	if err != errAuthCodeExpired {
		t.Fatalf("expected errAuthCodeExpired, got %v", err)
	}
}

func TestDeleteAuthCode(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	if err := store.SaveAuthCode(&Code{
		Code:      "delete-me",
		ClientID:  "client-1",
		ExpiresAt: now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("saving auth code: %v", err)
	}

	store.DeleteAuthCode("delete-me")

	_, err := store.GetAuthCode("delete-me", now)
	if err != errAuthCodeNotFound {
		t.Fatalf("expected errAuthCodeNotFound after delete, got %v", err)
	}
}

func TestDeleteAuthCode_Nonexistent(t *testing.T) {
	t.Parallel()

	store := NewStore()

	// Should not panic on nonexistent key.
	store.DeleteAuthCode("does-not-exist")
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

	if err := store.SaveRefreshToken(token); err != nil {
		t.Fatalf("saving refresh token: %v", err)
	}

	got, err := store.ConsumeRefreshToken("refresh-token-1", "client-1", now)
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

	if err := store.SaveRefreshToken(token); err != nil {
		t.Fatalf("saving refresh token: %v", err)
	}

	_, err := store.ConsumeRefreshToken("old-refresh-token", "client-1", now)
	if err != nil {
		t.Fatalf("first consume should succeed: %v", err)
	}

	_, err = store.ConsumeRefreshToken("old-refresh-token", "client-1", now)
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

	if err = store.SaveRefreshToken(newToken); err != nil {
		t.Fatalf("saving new refresh token: %v", err)
	}

	got, err := store.ConsumeRefreshToken("new-refresh-token", "client-1", now)
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

	if err := store.SaveRefreshToken(token); err != nil {
		t.Fatalf("saving refresh token: %v", err)
	}

	_, err := store.ConsumeRefreshToken("expired-refresh", "client-1", time.Now())
	if err == nil {
		t.Fatal("expected error for expired refresh token")
	}

	if err != errRefreshTokenNotFound {
		t.Fatalf("expected errRefreshTokenNotFound for expired token (uniform error), got %v", err)
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

	if err := store.SaveRefreshToken(token); err != nil {
		t.Fatalf("saving refresh token: %v", err)
	}

	_, _ = store.ConsumeRefreshToken("expired-deleted-refresh", "client-1", time.Now())

	_, err := store.ConsumeRefreshToken("expired-deleted-refresh", "client-1", time.Now())
	if err != errRefreshTokenNotFound {
		t.Fatalf("expected errRefreshTokenNotFound after consuming expired token, got %v", err)
	}
}

func TestConsumeRefreshToken_NotFound(t *testing.T) {
	t.Parallel()

	store := NewStore()

	_, err := store.ConsumeRefreshToken("nonexistent", "any-client", time.Now())
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

	if err := store.SaveClient(client); err != nil {
		t.Fatalf("unexpected error saving client: %v", err)
	}

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

	if err := store.SaveClient(client1); err != nil {
		t.Fatalf("unexpected error saving client1: %v", err)
	}

	client2 := &RegisteredClient{
		ClientID:   "client-overwrite",
		ClientName: "Updated",
		CreatedAt:  time.Now(),
	}

	if err := store.SaveClient(client2); err != nil {
		t.Fatalf("unexpected error saving client2: %v", err)
	}

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

	_, err := store.ValidateAndConsumeAuthCode("any", time.Now(), noValidation)
	if err != errAuthCodeNotFound {
		t.Fatalf("expected errAuthCodeNotFound on empty store, got %v", err)
	}

	_, err = store.ConsumeRefreshToken("any", "any-client", time.Now())
	if err != errRefreshTokenNotFound {
		t.Fatalf("expected errRefreshTokenNotFound on empty store, got %v", err)
	}

	_, err = store.GetClient("any")
	if err != errClientNotFound {
		t.Fatalf("expected errClientNotFound on empty store, got %v", err)
	}
}

func TestSaveClient_MaxClientsLimit(t *testing.T) {
	t.Parallel()

	store := NewStore()

	for i := range maxClients {
		err := store.SaveClient(&RegisteredClient{
			ClientID:  fmt.Sprintf("client-%d", i),
			CreatedAt: time.Now(),
		})
		if err != nil {
			t.Fatalf("unexpected error saving client %d: %v", i, err)
		}
	}

	err := store.SaveClient(&RegisteredClient{
		ClientID:  "one-too-many",
		CreatedAt: time.Now(),
	})
	if err != errMaxClientsReached {
		t.Fatalf("expected errMaxClientsReached, got %v", err)
	}
}

func TestEvictExpired(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	if err := store.SaveAuthCode(&Code{
		Code:      "expired-code",
		ClientID:  "client-1",
		ExpiresAt: now.Add(-5 * time.Minute),
	}); err != nil {
		t.Fatalf("saving expired auth code: %v", err)
	}

	if err := store.SaveAuthCode(&Code{
		Code:      "valid-code",
		ClientID:  "client-1",
		ExpiresAt: now.Add(5 * time.Minute),
	}); err != nil {
		t.Fatalf("saving valid auth code: %v", err)
	}

	if err := store.SaveRefreshToken(&RefreshToken{
		Token:     "expired-rt",
		ClientID:  "client-1",
		ExpiresAt: now.Add(-1 * time.Hour),
	}); err != nil {
		t.Fatalf("saving expired refresh token: %v", err)
	}

	if err := store.SaveRefreshToken(&RefreshToken{
		Token:     "valid-rt",
		ClientID:  "client-1",
		ExpiresAt: now.Add(1 * time.Hour),
	}); err != nil {
		t.Fatalf("saving valid refresh token: %v", err)
	}

	store.evictExpired(now)

	_, err := store.GetAuthCode("expired-code", now)
	if err != errAuthCodeNotFound {
		t.Fatalf("expected expired code to be evicted, got %v", err)
	}

	got, err := store.GetAuthCode("valid-code", now)
	if err != nil {
		t.Fatalf("expected valid code to survive eviction: %v", err)
	}

	if got.Code != "valid-code" {
		t.Fatalf("expected 'valid-code', got %q", got.Code)
	}

	_, err = store.ConsumeRefreshToken("expired-rt", "client-1", now)
	if err != errRefreshTokenNotFound {
		t.Fatalf("expected expired refresh token to be evicted, got %v", err)
	}

	gotRT, err := store.ConsumeRefreshToken("valid-rt", "client-1", now)
	if err != nil {
		t.Fatalf("expected valid refresh token to survive eviction: %v", err)
	}

	if gotRT.Token != "valid-rt" {
		t.Fatalf("expected 'valid-rt', got %q", gotRT.Token)
	}
}

func TestEvictExpired_EvictsExpiredClients(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	if err := store.SaveClient(&RegisteredClient{
		ClientID:  "old-client",
		CreatedAt: now.Add(-(clientTTL + time.Hour)),
	}); err != nil {
		t.Fatalf("saving old client: %v", err)
	}

	if err := store.SaveClient(&RegisteredClient{
		ClientID:  "fresh-client",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("saving fresh client: %v", err)
	}

	store.evictExpired(now)

	_, err := store.GetClient("old-client")
	if err != errClientNotFound {
		t.Fatalf("expected expired client to be evicted, got %v", err)
	}

	got, err := store.GetClient("fresh-client")
	if err != nil {
		t.Fatalf("expected fresh client to survive eviction: %v", err)
	}

	if got.ClientID != "fresh-client" {
		t.Fatalf("expected 'fresh-client', got %q", got.ClientID)
	}
}

func TestSaveClient_AutoEvictsExpiredOnCapHit(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	for i := range maxClients {
		if err := store.SaveClient(&RegisteredClient{
			ClientID:  fmt.Sprintf("old-%d", i),
			CreatedAt: now.Add(-(clientTTL + time.Hour)),
		}); err != nil {
			t.Fatalf("saving client %d: %v", i, err)
		}
	}

	err := store.SaveClient(&RegisteredClient{
		ClientID:  "auto-evicted",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("expected auto-eviction to free slots, got %v", err)
	}

	got, err := store.GetClient("auto-evicted")
	if err != nil {
		t.Fatalf("expected new client to exist: %v", err)
	}

	if got.ClientID != "auto-evicted" {
		t.Fatalf("expected 'auto-evicted', got %q", got.ClientID)
	}
}

func TestSaveAuthCode_MaxAuthCodesLimit(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	for i := range maxAuthCodes {
		err := store.SaveAuthCode(&Code{
			Code:      fmt.Sprintf("code-%d", i),
			ClientID:  "client-1",
			ExpiresAt: now.Add(10 * time.Minute),
		})
		if err != nil {
			t.Fatalf("unexpected error saving auth code %d: %v", i, err)
		}
	}

	err := store.SaveAuthCode(&Code{
		Code:      "one-too-many",
		ClientID:  "client-1",
		ExpiresAt: now.Add(10 * time.Minute),
	})
	if err != errMaxAuthCodesReached {
		t.Fatalf("expected errMaxAuthCodesReached, got %v", err)
	}
}

func TestSaveAuthCode_AutoEvictsExpiredOnCapHit(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	for i := range maxAuthCodes {
		err := store.SaveAuthCode(&Code{
			Code:      fmt.Sprintf("expired-code-%d", i),
			ClientID:  "client-1",
			ExpiresAt: now.Add(-5 * time.Minute),
		})
		if err != nil {
			t.Fatalf("saving auth code %d: %v", i, err)
		}
	}

	err := store.SaveAuthCode(&Code{
		Code:      "fresh-code",
		ClientID:  "client-1",
		ExpiresAt: now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("expected auto-eviction to free slots, got %v", err)
	}

	got, err := store.GetAuthCode("fresh-code", now)
	if err != nil {
		t.Fatalf("expected new code to exist: %v", err)
	}

	if got.Code != "fresh-code" {
		t.Fatalf("expected 'fresh-code', got %q", got.Code)
	}
}

func TestSaveRefreshToken_MaxRefreshTokensLimit(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	for i := range maxRefreshTokens {
		err := store.SaveRefreshToken(&RefreshToken{
			Token:     fmt.Sprintf("rt-%d", i),
			ClientID:  "client-1",
			ExpiresAt: now.Add(24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("unexpected error saving refresh token %d: %v", i, err)
		}
	}

	err := store.SaveRefreshToken(&RefreshToken{
		Token:     "one-too-many",
		ClientID:  "client-1",
		ExpiresAt: now.Add(24 * time.Hour),
	})
	if err != ErrMaxRefreshTokensReached {
		t.Fatalf("expected ErrMaxRefreshTokensReached, got %v", err)
	}
}

func TestSaveRefreshToken_AutoEvictsExpiredOnCapHit(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	for i := range maxRefreshTokens {
		err := store.SaveRefreshToken(&RefreshToken{
			Token:     fmt.Sprintf("expired-rt-%d", i),
			ClientID:  "client-1",
			ExpiresAt: now.Add(-1 * time.Hour),
		})
		if err != nil {
			t.Fatalf("saving refresh token %d: %v", i, err)
		}
	}

	err := store.SaveRefreshToken(&RefreshToken{
		Token:     "fresh-rt",
		ClientID:  "client-1",
		ExpiresAt: now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("expected auto-eviction to free slots, got %v", err)
	}

	got, err := store.ConsumeRefreshToken("fresh-rt", "client-1", now)
	if err != nil {
		t.Fatalf("expected new token to exist: %v", err)
	}

	if got.Token != "fresh-rt" {
		t.Fatalf("expected 'fresh-rt', got %q", got.Token)
	}
}

func TestConsumeRefreshToken_ExpiredAndNotFoundReturnSameError(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Now()

	if err := store.SaveRefreshToken(&RefreshToken{
		Token:     "expired-uniform",
		ClientID:  "client-1",
		ExpiresAt: now.Add(-1 * time.Hour),
	}); err != nil {
		t.Fatalf("saving refresh token: %v", err)
	}

	_, errExpired := store.ConsumeRefreshToken("expired-uniform", "client-1", now)
	_, errNotFound := store.ConsumeRefreshToken("nonexistent-uniform", "client-1", now)

	if errExpired != errNotFound {
		t.Fatalf("expected expired and not-found to return same error, got expired=%v, not-found=%v", errExpired, errNotFound)
	}

	if errExpired != errRefreshTokenNotFound {
		t.Fatalf("expected errRefreshTokenNotFound, got %v", errExpired)
	}
}

func TestStopCleanup_Idempotent(t *testing.T) {
	t.Parallel()

	store := NewStore()
	store.StartCleanup()

	store.StopCleanup()
	store.StopCleanup()
}
