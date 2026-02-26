package rest

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/0xalexb/intervals-icu-mcp/src/app/auth"
	"github.com/0xalexb/intervals-icu-mcp/src/app/clients/github"
)

const (
	testJWTSecret  = auth.JWTSecret("test-handler-jwt-secret-key-1234")
	testIssuer     = auth.Issuer("https://auth.example.com")
	testGHClientID = auth.GitHubClientID("gh-client-id")
	testGHSecret   = auth.GitHubClientSecret("gh-client-secret")
)

func newTestHandler(ghTokenURL, ghUserURL string) *Handler {
	return &Handler{
		store:          auth.NewStore(),
		allowedUsers:   auth.AllowedUsers{"alice", "bob"},
		ghClientID:     testGHClientID,
		ghClientSecret: testGHSecret,
		jwtSecret:      testJWTSecret,
		issuer:         testIssuer,
		metadata:       auth.NewAuthorizationServerMetadata(testIssuer),
		ghClient:       github.NewTestClient(ghTokenURL, ghUserURL),
	}
}

func newTestHandlerNoAllowList(ghTokenURL, ghUserURL string) *Handler {
	h := newTestHandler(ghTokenURL, ghUserURL)
	h.allowedUsers = auth.AllowedUsers{}

	return h
}

func computeS256Challenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))

	return base64.RawURLEncoding.EncodeToString(h[:])
}

func saveTestClient(t *testing.T, store *auth.Store, clientID string, grantTypes []string) {
	t.Helper()

	if err := store.SaveClient(&auth.RegisteredClient{
		ClientID:     clientID,
		RedirectURIs: []string{"https://client.example.com/callback"},
		ClientName:   "Test Client",
		GrantTypes:   grantTypes,
		CreatedAt:    time.Now(),
	}); err != nil {
		t.Fatalf("saving test client: %v", err)
	}
}

func TestHandleAuthServerMetadata(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", http.NoBody)

	h.HandleAuthServerMetadata(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	var meta auth.AuthorizationServerMetadata
	if err := json.Unmarshal(rec.Body.Bytes(), &meta); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if meta.Issuer != string(testIssuer) {
		t.Fatalf("expected issuer %q, got %q", testIssuer, meta.Issuer)
	}

	if meta.AuthorizationEndpoint != string(testIssuer)+"/oauth/authorize" {
		t.Fatalf("unexpected authorization_endpoint: %q", meta.AuthorizationEndpoint)
	}

	if meta.TokenEndpoint != string(testIssuer)+"/oauth/token" {
		t.Fatalf("unexpected token_endpoint: %q", meta.TokenEndpoint)
	}
}

func TestHandleAuthorize_ValidRequest(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	saveTestClient(t, h.store, "my-client", []string{"authorization_code"})

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {"my-client"},
		"redirect_uri":          {"https://client.example.com/callback"},
		"code_challenge":        {"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
		"code_challenge_method": {"S256"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), http.NoBody)

	h.HandleAuthorize(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d: %s", rec.Code, rec.Body.String())
	}

	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parsing Location header: %v", err)
	}

	if loc.Host != "github.com" {
		t.Fatalf("expected redirect to github.com, got %q", loc.Host)
	}

	if loc.Path != "/login/oauth/authorize" {
		t.Fatalf("expected path /login/oauth/authorize, got %q", loc.Path)
	}

	if loc.Query().Get("client_id") != string(testGHClientID) {
		t.Fatalf("expected client_id %q, got %q", testGHClientID, loc.Query().Get("client_id"))
	}

	if loc.Query().Get("state") == "" {
		t.Fatal("expected non-empty state parameter in redirect")
	}
}

func TestHandleAuthorize_UnregisteredClient(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {"unknown-client"},
		"redirect_uri":          {"https://evil.example.com/callback"},
		"code_challenge":        {"challenge"},
		"code_challenge_method": {"S256"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), http.NoBody)

	h.HandleAuthorize(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_request")
}

func TestHandleAuthorize_RedirectURIMismatch(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	saveTestClient(t, h.store, "my-client", []string{"authorization_code"})

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {"my-client"},
		"redirect_uri":          {"https://evil.example.com/steal"},
		"code_challenge":        {"challenge"},
		"code_challenge_method": {"S256"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), http.NoBody)

	h.HandleAuthorize(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_request")
}

func TestHandleAuthorize_MissingResponseType(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	params := url.Values{
		"client_id":             {"my-client"},
		"redirect_uri":          {"https://client.example.com/callback"},
		"code_challenge":        {"challenge"},
		"code_challenge_method": {"S256"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), http.NoBody)

	h.HandleAuthorize(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "unsupported_response_type")
}

func TestHandleAuthorize_MissingClientID(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	params := url.Values{
		"response_type":         {"code"},
		"redirect_uri":          {"https://client.example.com/callback"},
		"code_challenge":        {"challenge"},
		"code_challenge_method": {"S256"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), http.NoBody)

	h.HandleAuthorize(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_request")
}

func TestHandleAuthorize_MissingCodeChallenge(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {"my-client"},
		"redirect_uri":          {"https://client.example.com/callback"},
		"code_challenge_method": {"S256"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), http.NoBody)

	h.HandleAuthorize(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_request")
}

func TestHandleAuthorize_WrongCodeChallengeMethod(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {"my-client"},
		"redirect_uri":          {"https://client.example.com/callback"},
		"code_challenge":        {"challenge"},
		"code_challenge_method": {"plain"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), http.NoBody)

	h.HandleAuthorize(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_request")
}

func TestHandleCallback_SuccessWithAllowList(t *testing.T) {
	t.Parallel()

	ghTokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"access_token": "gho_test"})
	}))
	defer ghTokenSrv.Close()

	ghUserSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"login": "alice"})
	}))
	defer ghUserSrv.Close()

	h := newTestHandler(ghTokenSrv.URL, ghUserSrv.URL)

	as := authorizeState{
		ClientID:            "my-client",
		RedirectURI:         "https://client.example.com/callback",
		CodeChallenge:       "test-challenge",
		CodeChallengeMethod: "S256",
		OriginalState:       "client-state",
		Scope:               "mcp",
		Nonce:               "test-nonce",
		CreatedAt:           time.Now().Unix(),
	}

	signedState, err := h.signState(as)
	if err != nil {
		t.Fatalf("signing state: %v", err)
	}

	params := url.Values{
		"code":  {"github-code-123"},
		"state": {signedState},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/callback?"+params.Encode(), http.NoBody)

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d: %s", rec.Code, rec.Body.String())
	}

	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parsing Location: %v", err)
	}

	if loc.Host != "client.example.com" {
		t.Fatalf("expected redirect to client.example.com, got %q", loc.Host)
	}

	if loc.Query().Get("code") == "" {
		t.Fatal("expected non-empty code in redirect")
	}

	if loc.Query().Get("state") != "client-state" {
		t.Fatalf("expected state 'client-state', got %q", loc.Query().Get("state"))
	}
}

func TestHandleCallback_UserNotAllowed(t *testing.T) {
	t.Parallel()

	ghTokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"access_token": "gho_test"})
	}))
	defer ghTokenSrv.Close()

	ghUserSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"login": "evil-user"})
	}))
	defer ghUserSrv.Close()

	h := newTestHandler(ghTokenSrv.URL, ghUserSrv.URL)

	as := authorizeState{
		ClientID:    "my-client",
		RedirectURI: "https://client.example.com/callback",
		Nonce:       "nonce",
		CreatedAt:   time.Now().Unix(),
	}

	signedState, err := h.signState(as)
	if err != nil {
		t.Fatalf("signing state: %v", err)
	}

	params := url.Values{
		"code":  {"github-code"},
		"state": {signedState},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/callback?"+params.Encode(), http.NoBody)

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "access_denied")
}

func TestHandleCallback_EmptyAllowList(t *testing.T) {
	t.Parallel()

	ghTokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"access_token": "gho_test"})
	}))
	defer ghTokenSrv.Close()

	ghUserSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"login": "anyone"})
	}))
	defer ghUserSrv.Close()

	h := newTestHandlerNoAllowList(ghTokenSrv.URL, ghUserSrv.URL)

	as := authorizeState{
		ClientID:    "my-client",
		RedirectURI: "https://client.example.com/callback",
		Nonce:       "nonce",
		CreatedAt:   time.Now().Unix(),
	}

	signedState, err := h.signState(as)
	if err != nil {
		t.Fatalf("signing state: %v", err)
	}

	params := url.Values{
		"code":  {"github-code"},
		"state": {signedState},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/callback?"+params.Encode(), http.NoBody)

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 (any user allowed), got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCallback_InvalidState(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	params := url.Values{
		"code":  {"github-code"},
		"state": {"tampered-state"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/callback?"+params.Encode(), http.NoBody)

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_request")
}

func TestHandleCallback_MissingCode(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	params := url.Values{
		"state": {"some-state"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/callback?"+params.Encode(), http.NoBody)

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleCallback_GitHubError(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	params := url.Values{
		"error":             {"access_denied"},
		"error_description": {"user denied access"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/callback?"+params.Encode(), http.NoBody)

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "access_denied")
}

func TestHandleToken_AuthorizationCodeGrant(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")
	saveTestClient(t, h.store, "my-client", []string{"authorization_code", "refresh_token"})

	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	codeChallenge := computeS256Challenge(codeVerifier)

	h.store.SaveAuthCode(&auth.Code{
		Code:                "test-auth-code",
		ClientID:            "my-client",
		RedirectURI:         "https://client.example.com/callback",
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
		GitHubUsername:      "alice",
		Scopes:              []string{"mcp"},
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"test-auth-code"},
		"code_verifier": {codeVerifier},
		"client_id":     {"my-client"},
		"redirect_uri":  {"https://client.example.com/callback"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp tokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if resp.AccessToken == "" {
		t.Fatal("expected non-empty access_token")
	}

	if resp.TokenType != "Bearer" {
		t.Fatalf("expected token_type 'Bearer', got %q", resp.TokenType)
	}

	if resp.RefreshToken == "" {
		t.Fatal("expected non-empty refresh_token")
	}

	if resp.ExpiresIn <= 0 {
		t.Fatalf("expected positive expires_in, got %d", resp.ExpiresIn)
	}

	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("expected Cache-Control: no-store, got %q", cc)
	}

	verifier := auth.NewTokenVerifier(testJWTSecret, testIssuer)

	info, err := verifier(context.Background(), resp.AccessToken, nil)
	if err != nil {
		t.Fatalf("verifying access token: %v", err)
	}

	if info.UserID != "alice" {
		t.Fatalf("expected UserID 'alice', got %q", info.UserID)
	}
}

func TestHandleToken_AuthorizationCodeGrant_WrongVerifier(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	codeChallenge := computeS256Challenge("correct-verifier")

	h.store.SaveAuthCode(&auth.Code{
		Code:                "test-code",
		ClientID:            "my-client",
		RedirectURI:         "https://client.example.com/callback",
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
		GitHubUsername:      "alice",
		Scopes:              []string{"mcp"},
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"test-code"},
		"code_verifier": {"wrong-verifier"},
		"client_id":     {"my-client"},
		"redirect_uri":  {"https://client.example.com/callback"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_grant")
}

func TestHandleToken_AuthorizationCodeGrant_ExpiredCode(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	h.store.SaveAuthCode(&auth.Code{
		Code:                "expired-code",
		ClientID:            "my-client",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		GitHubUsername:      "alice",
		Scopes:              []string{"mcp"},
		ExpiresAt:           time.Now().Add(-1 * time.Minute),
	})

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"expired-code"},
		"code_verifier": {"verifier"},
		"client_id":     {"my-client"},
		"redirect_uri":  {"https://example.com/callback"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_grant")
}

func TestHandleToken_AuthorizationCodeGrant_CodeReuse(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")
	saveTestClient(t, h.store, "my-client", []string{"authorization_code", "refresh_token"})

	codeVerifier := "my-verifier"
	codeChallenge := computeS256Challenge(codeVerifier)

	h.store.SaveAuthCode(&auth.Code{
		Code:                "one-time-code",
		ClientID:            "my-client",
		RedirectURI:         "https://client.example.com/callback",
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
		GitHubUsername:      "alice",
		Scopes:              []string{"mcp"},
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"one-time-code"},
		"code_verifier": {codeVerifier},
		"client_id":     {"my-client"},
		"redirect_uri":  {"https://client.example.com/callback"},
	}

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first exchange: expected 200, got %d", rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("second exchange: expected 400 (code reuse), got %d", rec2.Code)
	}
}

func TestHandleToken_AuthorizationCodeGrant_ClientIDMismatch(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	codeVerifier := "verifier"
	codeChallenge := computeS256Challenge(codeVerifier)

	h.store.SaveAuthCode(&auth.Code{
		Code:                "test-code",
		ClientID:            "original-client",
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
		GitHubUsername:      "alice",
		Scopes:              []string{"mcp"},
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"test-code"},
		"code_verifier": {codeVerifier},
		"client_id":     {"different-client"},
		"redirect_uri":  {"https://example.com/callback"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_grant")
}

func TestHandleToken_RefreshTokenGrant(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")
	saveTestClient(t, h.store, "my-client", []string{"authorization_code", "refresh_token"})

	h.store.SaveRefreshToken(&auth.RefreshToken{
		Token:         "test-refresh-token",
		ClientID:      "my-client",
		GitHubUsername: "alice",
		Scopes:        []string{"mcp"},
		ExpiresAt:     time.Now().Add(30 * 24 * time.Hour),
	})

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"test-refresh-token"},
		"client_id":     {"my-client"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp tokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if resp.AccessToken == "" {
		t.Fatal("expected non-empty access_token")
	}

	if resp.RefreshToken == "" {
		t.Fatal("expected non-empty refresh_token")
	}

	if resp.RefreshToken == "test-refresh-token" {
		t.Fatal("expected rotated refresh token (different from original)")
	}
}

func TestHandleToken_RefreshTokenGrant_TokenRotation(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")
	saveTestClient(t, h.store, "my-client", []string{"authorization_code", "refresh_token"})

	h.store.SaveRefreshToken(&auth.RefreshToken{
		Token:         "original-rt",
		ClientID:      "my-client",
		GitHubUsername: "alice",
		Scopes:        []string{"mcp"},
		ExpiresAt:     time.Now().Add(30 * 24 * time.Hour),
	})

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"original-rt"},
		"client_id":     {"my-client"},
	}

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first refresh: expected 200, got %d", rec1.Code)
	}

	form2 := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"original-rt"},
		"client_id":     {"my-client"},
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form2.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("second refresh: expected 400 (token consumed), got %d", rec2.Code)
	}
}

func TestHandleToken_RefreshTokenGrant_ExpiredToken(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")
	saveTestClient(t, h.store, "my-client", []string{"authorization_code", "refresh_token"})

	h.store.SaveRefreshToken(&auth.RefreshToken{
		Token:         "expired-rt",
		ClientID:      "my-client",
		GitHubUsername: "alice",
		Scopes:        []string{"mcp"},
		ExpiresAt:     time.Now().Add(-1 * time.Hour),
	})

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"expired-rt"},
		"client_id":     {"my-client"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_grant")
}

func TestHandleToken_UnsupportedGrantType(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	form := url.Values{
		"grant_type": {"client_credentials"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "unsupported_grant_type")
}

func TestHandleToken_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/token", http.NoBody)

	h.HandleToken(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleRegister_Success(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	body := `{"redirect_uris": ["https://client.example.com/callback"], "client_name": "Test App"}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	h.HandleRegister(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp registrationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if resp.ClientID == "" {
		t.Fatal("expected non-empty client_id")
	}

	if len(resp.RedirectURIs) != 1 || resp.RedirectURIs[0] != "https://client.example.com/callback" {
		t.Fatalf("unexpected redirect_uris: %v", resp.RedirectURIs)
	}

	if resp.ClientName != "Test App" {
		t.Fatalf("expected client_name 'Test App', got %q", resp.ClientName)
	}

	if len(resp.GrantTypes) != 1 || resp.GrantTypes[0] != "authorization_code" {
		t.Fatalf("expected default grant_types [authorization_code], got %v", resp.GrantTypes)
	}

	client, err := h.store.GetClient(resp.ClientID)
	if err != nil {
		t.Fatalf("client not found in store: %v", err)
	}

	if client.ClientName != "Test App" {
		t.Fatalf("stored client name mismatch: %q", client.ClientName)
	}
}

func TestHandleRegister_CustomGrantTypes(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	body := `{"redirect_uris": ["https://client.example.com/callback"], "grant_types": ["authorization_code", "refresh_token"]}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	h.HandleRegister(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var resp registrationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if len(resp.GrantTypes) != 2 {
		t.Fatalf("expected 2 grant types, got %d", len(resp.GrantTypes))
	}
}

func TestHandleRegister_MissingRedirectURIs(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	body := `{"client_name": "Test"}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	h.HandleRegister(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_client_metadata")
}

func TestHandleRegister_InvalidRedirectURI(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	body := `{"redirect_uris": ["not a valid uri"]}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	h.HandleRegister(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_client_metadata")
}

func TestHandleRegister_RelativeRedirectURI(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	body := `{"redirect_uris": ["/callback"]}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	h.HandleRegister(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for relative URI, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_client_metadata")
}

func TestHandleRegister_JavascriptRedirectURI(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	body := `{"redirect_uris": ["javascript:alert(1)"]}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	h.HandleRegister(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for javascript: URI, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_client_metadata")
}

func TestHandleRegister_FragmentInRedirectURI(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	body := `{"redirect_uris": ["https://example.com/callback#frag"]}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	h.HandleRegister(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for URI with fragment, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_client_metadata")
}

func TestHandleRegister_InvalidJSON(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")

	h.HandleRegister(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleRegister_HTTPNonLoopbackRedirectURI(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	body := `{"redirect_uris": ["http://external.example.com/callback"]}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	h.HandleRegister(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-loopback http redirect_uri, got %d: %s", rec.Code, rec.Body.String())
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_client_metadata")
}

func TestHandleRegister_HTTPLocalhostRedirectURI(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	body := `{"redirect_uris": ["http://localhost:8080/callback"]}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	h.HandleRegister(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for loopback http redirect_uri, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleToken_RefreshTokenGrant_ClientWithoutRefreshGrant(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")
	saveTestClient(t, h.store, "auth-only-client", []string{"authorization_code"})

	h.store.SaveRefreshToken(&auth.RefreshToken{
		Token:         "some-refresh-token",
		ClientID:      "auth-only-client",
		GitHubUsername: "alice",
		Scopes:        []string{"mcp"},
		ExpiresAt:     time.Now().Add(30 * 24 * time.Hour),
	})

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"some-refresh-token"},
		"client_id":     {"auth-only-client"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "unauthorized_client")
}

func TestHandleToken_AuthorizationCodeGrant_NoRefreshTokenWhenNotRegistered(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")
	saveTestClient(t, h.store, "auth-only-client", []string{"authorization_code"})

	codeVerifier := "verifier-for-no-refresh"
	codeChallenge := computeS256Challenge(codeVerifier)

	h.store.SaveAuthCode(&auth.Code{
		Code:                "code-no-refresh",
		ClientID:            "auth-only-client",
		RedirectURI:         "https://client.example.com/callback",
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
		GitHubUsername:      "alice",
		Scopes:              []string{"mcp"},
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"code-no-refresh"},
		"code_verifier": {codeVerifier},
		"client_id":     {"auth-only-client"},
		"redirect_uri":  {"https://client.example.com/callback"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp tokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if resp.AccessToken == "" {
		t.Fatal("expected non-empty access_token")
	}

	if resp.RefreshToken != "" {
		t.Fatalf("expected empty refresh_token for client without refresh_token grant, got %q", resp.RefreshToken)
	}
}

func TestHandleRegister_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/register", http.NoBody)

	h.HandleRegister(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestFullOAuthFlow(t *testing.T) {
	t.Parallel()

	ghTokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"access_token": "gho_flow_token"})
	}))
	defer ghTokenSrv.Close()

	ghUserSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"login": "alice"})
	}))
	defer ghUserSrv.Close()

	h := newTestHandler(ghTokenSrv.URL, ghUserSrv.URL)

	// Step 1: Register client (with both grant types to test refresh flow)
	regBody := `{"redirect_uris": ["https://client.example.com/callback"], "client_name": "Flow Test", "grant_types": ["authorization_code", "refresh_token"]}`

	regRec := httptest.NewRecorder()
	regReq := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")

	h.HandleRegister(regRec, regReq)

	if regRec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", regRec.Code, regRec.Body.String())
	}

	var regResp registrationResponse
	if err := json.Unmarshal(regRec.Body.Bytes(), &regResp); err != nil {
		t.Fatalf("register: decoding response: %v", err)
	}

	clientID := regResp.ClientID

	// Step 2: Authorize with PKCE
	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk-flow-test"
	codeChallenge := computeS256Challenge(codeVerifier)

	authParams := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {"https://client.example.com/callback"},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {"flow-state-123"},
		"scope":                 {"mcp"},
	}

	authRec := httptest.NewRecorder()
	authReq := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+authParams.Encode(), http.NoBody)

	h.HandleAuthorize(authRec, authReq)

	if authRec.Code != http.StatusFound {
		t.Fatalf("authorize: expected 302, got %d: %s", authRec.Code, authRec.Body.String())
	}

	ghRedirect, err := url.Parse(authRec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("authorize: parsing Location: %v", err)
	}
	signedState := ghRedirect.Query().Get("state")

	// Step 3: Callback (simulating GitHub redirect back)
	cbParams := url.Values{
		"code":  {"github-auth-code"},
		"state": {signedState},
	}

	cbRec := httptest.NewRecorder()
	cbReq := httptest.NewRequest(http.MethodGet, "/oauth/callback?"+cbParams.Encode(), http.NoBody)

	h.HandleCallback(cbRec, cbReq)

	if cbRec.Code != http.StatusFound {
		t.Fatalf("callback: expected 302, got %d: %s", cbRec.Code, cbRec.Body.String())
	}

	clientRedirect, err := url.Parse(cbRec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("callback: parsing Location: %v", err)
	}
	authCode := clientRedirect.Query().Get("code")

	if authCode == "" {
		t.Fatal("callback: expected non-empty auth code")
	}

	if clientRedirect.Query().Get("state") != "flow-state-123" {
		t.Fatalf("callback: expected state 'flow-state-123', got %q", clientRedirect.Query().Get("state"))
	}

	// Step 4: Token exchange
	tokenForm := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {authCode},
		"code_verifier": {codeVerifier},
		"client_id":     {clientID},
		"redirect_uri":  {"https://client.example.com/callback"},
	}

	tokenRec := httptest.NewRecorder()
	tokenReq := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(tokenForm.Encode()))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(tokenRec, tokenReq)

	if tokenRec.Code != http.StatusOK {
		t.Fatalf("token: expected 200, got %d: %s", tokenRec.Code, tokenRec.Body.String())
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(tokenRec.Body.Bytes(), &tokenResp); err != nil {
		t.Fatalf("token: decoding response: %v", err)
	}

	if tokenResp.AccessToken == "" {
		t.Fatal("token: expected non-empty access_token")
	}

	if tokenResp.RefreshToken == "" {
		t.Fatal("token: expected non-empty refresh_token")
	}

	// Verify the access token
	verifier := auth.NewTokenVerifier(testJWTSecret, testIssuer)

	info, err := verifier(context.Background(), tokenResp.AccessToken, nil)
	if err != nil {
		t.Fatalf("verifying access token: %v", err)
	}

	if info.UserID != "alice" {
		t.Fatalf("expected UserID 'alice', got %q", info.UserID)
	}

	if len(info.Scopes) != 1 || info.Scopes[0] != "mcp" {
		t.Fatalf("expected scopes [mcp], got %v", info.Scopes)
	}

	// Step 5: Refresh token
	refreshForm := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {tokenResp.RefreshToken},
		"client_id":     {clientID},
	}

	refreshRec := httptest.NewRecorder()
	refreshReq := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(refreshForm.Encode()))
	refreshReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(refreshRec, refreshReq)

	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh: expected 200, got %d: %s", refreshRec.Code, refreshRec.Body.String())
	}

	var refreshResp tokenResponse
	if err := json.Unmarshal(refreshRec.Body.Bytes(), &refreshResp); err != nil {
		t.Fatalf("refresh: decoding response: %v", err)
	}

	if refreshResp.AccessToken == "" {
		t.Fatal("refresh: expected non-empty access_token")
	}

	if refreshResp.RefreshToken == "" {
		t.Fatal("refresh: expected non-empty refresh_token")
	}

	if refreshResp.RefreshToken == tokenResp.RefreshToken {
		t.Fatal("refresh: expected rotated refresh token")
	}
}

func TestSignVerifyState_RoundTrip(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	original := authorizeState{
		ClientID:            "test-client",
		RedirectURI:         "https://client.example.com/callback",
		CodeChallenge:       "challenge-value",
		CodeChallengeMethod: "S256",
		OriginalState:       "original-state",
		Scope:               "mcp read",
		Nonce:               "test-nonce",
		CreatedAt:           time.Now().Unix(),
	}

	signed, err := h.signState(original)
	if err != nil {
		t.Fatalf("signing state: %v", err)
	}

	recovered, err := h.verifyState(signed)
	if err != nil {
		t.Fatalf("verifying state: %v", err)
	}

	if recovered.ClientID != original.ClientID {
		t.Fatalf("ClientID mismatch: %q vs %q", recovered.ClientID, original.ClientID)
	}

	if recovered.RedirectURI != original.RedirectURI {
		t.Fatalf("RedirectURI mismatch: %q vs %q", recovered.RedirectURI, original.RedirectURI)
	}

	if recovered.OriginalState != original.OriginalState {
		t.Fatalf("OriginalState mismatch: %q vs %q", recovered.OriginalState, original.OriginalState)
	}
}

func TestVerifyState_TamperedPayload(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	as := authorizeState{
		ClientID:    "test-client",
		RedirectURI: "https://client.example.com/callback",
		Nonce:       "nonce",
		CreatedAt:   time.Now().Unix(),
	}

	signed, err := h.signState(as)
	if err != nil {
		t.Fatalf("signing state: %v", err)
	}

	parts := strings.SplitN(signed, ".", 2)
	tampered := "dGFtcGVyZWQ" + "." + parts[1]

	_, err = h.verifyState(tampered)
	if err == nil {
		t.Fatal("expected error for tampered state")
	}
}

func TestVerifyState_NoSeparator(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	_, err := h.verifyState("no-dot-separator")
	if err == nil {
		t.Fatal("expected error for state without separator")
	}
}

func TestVerifyCodeChallenge_Valid(t *testing.T) {
	t.Parallel()

	verifier := "test-code-verifier-123"
	challenge := computeS256Challenge(verifier)

	if !verifyCodeChallenge(challenge, verifier) {
		t.Fatal("expected valid code challenge verification")
	}
}

func TestVerifyCodeChallenge_Invalid(t *testing.T) {
	t.Parallel()

	challenge := computeS256Challenge("correct-verifier")

	if verifyCodeChallenge(challenge, "wrong-verifier") {
		t.Fatal("expected failed code challenge verification")
	}
}

func TestParseScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"  ", nil},
		{"mcp", []string{"mcp"}},
		{"mcp read write", []string{"mcp", "read", "write"}},
		{"  mcp  read  ", []string{"mcp", "read"}},
	}

	for _, tt := range tests {
		result := parseScopes(tt.input)
		if len(result) != len(tt.expected) {
			t.Fatalf("parseScopes(%q): expected %v, got %v", tt.input, tt.expected, result)
		}

		for i := range result {
			if result[i] != tt.expected[i] {
				t.Fatalf("parseScopes(%q)[%d]: expected %q, got %q", tt.input, i, tt.expected[i], result[i])
			}
		}
	}
}

func TestHandleToken_MissingCode(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code_verifier": {"verifier"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_request")
}

func TestHandleToken_MissingCodeVerifier(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	form := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {"test-code"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_request")
}

func TestHandleToken_MissingRefreshToken(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	form := url.Values{
		"grant_type": {"refresh_token"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "invalid_request")
}

func TestHandleAuthorize_ClientWithoutAuthCodeGrant(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	saveTestClient(t, h.store, "refresh-only-client", []string{"refresh_token"})

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {"refresh-only-client"},
		"redirect_uri":          {"https://client.example.com/callback"},
		"code_challenge":        {"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
		"code_challenge_method": {"S256"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), http.NoBody)

	h.HandleAuthorize(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "unauthorized_client")
}

func TestHandleToken_AuthorizationCodeGrant_ClientWithoutAuthCodeGrant(t *testing.T) {
	t.Parallel()

	h := newTestHandler("", "")

	// Register client with only refresh_token grant.
	saveTestClient(t, h.store, "refresh-only", []string{"refresh_token"})

	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	codeChallenge := computeS256Challenge(codeVerifier)

	// Manually save an auth code for this client (bypasses HandleAuthorize check).
	h.store.SaveAuthCode(&auth.Code{
		Code:                "code-for-refresh-only",
		ClientID:            "refresh-only",
		RedirectURI:         "https://client.example.com/callback",
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
		GitHubUsername:      "alice",
		Scopes:              []string{"mcp"},
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	})

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"code-for-refresh-only"},
		"code_verifier": {codeVerifier},
		"client_id":     {"refresh-only"},
		"redirect_uri":  {"https://client.example.com/callback"},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h.HandleToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	assertOAuthErrorCode(t, rec.Body.Bytes(), "unauthorized_client")
}

func assertOAuthErrorCode(t *testing.T, body []byte, expectedCode string) {
	t.Helper()

	var errResp oauthErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("decoding error response: %v (body: %s)", err, string(body))
	}

	if errResp.Error != expectedCode {
		t.Fatalf("expected error code %q, got %q (description: %s)", expectedCode, errResp.Error, errResp.Description)
	}
}
