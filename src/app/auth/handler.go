package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"go.uber.org/fx"
)

const (
	authCodeTTL        = 10 * time.Minute
	accessTokenTTL     = 1 * time.Hour
	refreshTokenTTL    = 30 * 24 * time.Hour
	stateTTL           = 10 * time.Minute
	stateNonceLength   = 16
	authCodeLength     = 32
	signedStateParts   = 2
	githubAuthorizeURL = "https://github.com/login/oauth/authorize"

	grantTypeAuthorizationCode = "authorization_code"
	grantTypeRefreshToken      = "refresh_token"

	schemeHTTP  = "http"
	schemeHTTPS = "https"

	scopeMCP = "mcp"
)

var errInvalidState = errors.New("invalid or tampered state parameter")

// HandlerParams holds the DI-injected dependencies for the OAuth Handler.
type HandlerParams struct {
	fx.In

	Store                       *Store
	AllowedUsers                AllowedUsers
	GitHubClientID              GitHubClientID
	GitHubClientSecret          GitHubClientSecret
	JWTSecret                   JWTSecret
	Issuer                      Issuer
	AuthorizationServerMetadata *AuthorizationServerMetadata
	GitHubClient                *GitHubClient
}

// Handler implements the OAuth 2.1 HTTP handlers.
type Handler struct {
	store          *Store
	allowedUsers   AllowedUsers
	ghClientID     GitHubClientID
	ghClientSecret GitHubClientSecret
	jwtSecret      JWTSecret
	issuer         Issuer
	metadata       *AuthorizationServerMetadata
	ghClient       *GitHubClient
}

// NewHandler creates an OAuth Handler from DI-injected dependencies.
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		store:          p.Store,
		allowedUsers:   p.AllowedUsers,
		ghClientID:     p.GitHubClientID,
		ghClientSecret: p.GitHubClientSecret,
		jwtSecret:      p.JWTSecret,
		issuer:         p.Issuer,
		metadata:       p.AuthorizationServerMetadata,
		ghClient:       p.GitHubClient,
	}
}

// authorizeState holds the original OAuth authorize parameters, serialized into
// the HMAC-signed state that is round-tripped through GitHub.
type authorizeState struct {
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	OriginalState       string `json:"original_state"`
	Scope               string `json:"scope"`
	Nonce               string `json:"nonce"`
	CreatedAt           int64  `json:"created_at"`
}

// oauthValidationError holds an OAuth error suitable for writing to the response.
type oauthValidationError struct {
	code        string
	description string
	status      int
}

func (e *oauthValidationError) Error() string {
	return e.description
}

// HandleAuthServerMetadata serves the OAuth 2.0 Authorization Server Metadata
// at GET /.well-known/oauth-authorization-server.
func (h *Handler) HandleAuthServerMetadata(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(h.metadata); err != nil {
		slog.Error("failed to encode metadata", "error", err)
	}
}

// HandleAuthorize handles GET /oauth/authorize. It validates PKCE parameters,
// generates an HMAC-signed state containing the original params, and redirects
// to GitHub's OAuth authorize endpoint.
func (h *Handler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	authState, valErr := validateAuthorizeParams(r.URL.Query())
	if valErr != nil {
		writeOAuthError(w, valErr.code, valErr.description, valErr.status)

		return
	}

	client, err := h.store.GetClient(authState.ClientID)
	if err != nil {
		writeOAuthError(w, "invalid_request", "unknown client_id", http.StatusBadRequest)

		return
	}

	if !slices.Contains(client.GrantTypes, grantTypeAuthorizationCode) {
		writeOAuthError(w, "unauthorized_client",
			"client is not registered for authorization_code grant", http.StatusBadRequest)

		return
	}

	if !slices.Contains(client.RedirectURIs, authState.RedirectURI) {
		writeOAuthError(w, "invalid_request", "redirect_uri not registered for this client", http.StatusBadRequest)

		return
	}

	signedState, err := h.signState(authState)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	ghURL, err := url.Parse(githubAuthorizeURL)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	ghParams := url.Values{
		"client_id":    {string(h.ghClientID)},
		"redirect_uri": {string(h.issuer) + "/oauth/callback"},
		"scope":        {"read:user"},
		"state":        {signedState},
	}

	ghURL.RawQuery = ghParams.Encode()
	http.Redirect(w, r, ghURL.String(), http.StatusFound)
}

func validateAuthorizeParams(query url.Values) (authorizeState, *oauthValidationError) {
	if query.Get("response_type") != "code" {
		return authorizeState{}, &oauthValidationError{
			"unsupported_response_type", "response_type must be 'code'", http.StatusBadRequest,
		}
	}

	clientID := query.Get("client_id")
	if clientID == "" {
		return authorizeState{}, &oauthValidationError{
			"invalid_request", "client_id is required", http.StatusBadRequest,
		}
	}

	redirectURI := query.Get("redirect_uri")
	if redirectURI == "" {
		return authorizeState{}, &oauthValidationError{
			"invalid_request", "redirect_uri is required", http.StatusBadRequest,
		}
	}

	codeChallenge := query.Get("code_challenge")
	if codeChallenge == "" {
		return authorizeState{}, &oauthValidationError{
			"invalid_request", "code_challenge is required (PKCE mandatory)", http.StatusBadRequest,
		}
	}

	if query.Get("code_challenge_method") != "S256" {
		return authorizeState{}, &oauthValidationError{
			"invalid_request", "code_challenge_method must be 'S256'", http.StatusBadRequest,
		}
	}

	nonce, err := generateRandomString(stateNonceLength)
	if err != nil {
		return authorizeState{}, &oauthValidationError{
			"server_error", "internal error", http.StatusInternalServerError,
		}
	}

	scope, scopeErr := validateScope(query.Get("scope"))
	if scopeErr != nil {
		return authorizeState{}, scopeErr
	}

	return authorizeState{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
		OriginalState:       query.Get("state"),
		Scope:               scope,
		Nonce:               nonce,
		CreatedAt:           time.Now().Unix(),
	}, nil
}

func validateScope(raw string) (string, *oauthValidationError) {
	scope := raw
	if scope == "" {
		scope = scopeMCP
	}

	for s := range strings.FieldsSeq(scope) {
		if s != scopeMCP {
			return "", &oauthValidationError{
				"invalid_scope", "unsupported scope: " + s, http.StatusBadRequest,
			}
		}
	}

	return scope, nil
}

// HandleCallback handles GET /oauth/callback. It validates the HMAC-signed state,
// exchanges the GitHub authorization code for an access token, fetches the GitHub user,
// checks the allowlist, generates an auth code, and redirects back to the client.
func (h *Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	code, authState, valErr := h.validateCallbackParams(r.URL.Query())
	if valErr != nil {
		writeOAuthError(w, valErr.code, valErr.description, valErr.status)

		return
	}

	ghUser, ghErr := h.resolveGitHubUser(r, code)
	if ghErr != nil {
		writeOAuthError(w, ghErr.code, ghErr.description, ghErr.status)

		return
	}

	authCode, err := generateRandomString(authCodeLength)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	h.store.SaveAuthCode(&Code{
		Code:                authCode,
		ClientID:            authState.ClientID,
		RedirectURI:         authState.RedirectURI,
		CodeChallenge:       authState.CodeChallenge,
		CodeChallengeMethod: authState.CodeChallengeMethod,
		GitHubUsername:      ghUser.Login,
		Scopes:              parseScopes(authState.Scope),
		ExpiresAt:           time.Now().Add(authCodeTTL),
	})

	redirectURL, err := url.Parse(authState.RedirectURI)
	if err != nil {
		http.Error(w, "invalid redirect URI", http.StatusInternalServerError)

		return
	}

	redirectQuery := redirectURL.Query()
	redirectQuery.Set("code", authCode)

	if authState.OriginalState != "" {
		redirectQuery.Set("state", authState.OriginalState)
	}

	redirectURL.RawQuery = redirectQuery.Encode()
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func (h *Handler) validateCallbackParams(
	query url.Values,
) (string, authorizeState, *oauthValidationError) {
	if errParam := query.Get("error"); errParam != "" {
		return "", authorizeState{}, &oauthValidationError{
			errParam, query.Get("error_description"), http.StatusBadRequest,
		}
	}

	code := query.Get("code")
	if code == "" {
		return "", authorizeState{}, &oauthValidationError{
			"invalid_request", "missing code parameter", http.StatusBadRequest,
		}
	}

	signedState := query.Get("state")
	if signedState == "" {
		return "", authorizeState{}, &oauthValidationError{
			"invalid_request", "missing state parameter", http.StatusBadRequest,
		}
	}

	authState, err := h.verifyState(signedState)
	if err != nil {
		return "", authorizeState{}, &oauthValidationError{
			"invalid_request", "invalid or tampered state", http.StatusBadRequest,
		}
	}

	return code, authState, nil
}

func (h *Handler) resolveGitHubUser(
	r *http.Request, code string,
) (*GitHubUser, *oauthValidationError) {
	ghToken, err := h.ghClient.ExchangeGitHubCode(r.Context(), h.ghClientID, h.ghClientSecret, code)
	if err != nil {
		return nil, &oauthValidationError{
			"server_error", "failed to exchange GitHub code", http.StatusInternalServerError,
		}
	}

	ghUser, err := h.ghClient.GetGitHubUser(r.Context(), ghToken)
	if err != nil {
		return nil, &oauthValidationError{
			"server_error", "failed to fetch GitHub user", http.StatusInternalServerError,
		}
	}

	if len(h.allowedUsers) > 0 && !h.allowedUsers.Contains(ghUser.Login) {
		return nil, &oauthValidationError{
			"access_denied", "user not authorized", http.StatusForbidden,
		}
	}

	return ghUser, nil
}

// HandleToken handles POST /oauth/token. It supports grant_type=authorization_code
// (with PKCE validation) and grant_type=refresh_token (with token rotation).
func (h *Handler) HandleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOAuthError(w, "invalid_request", "method must be POST", http.StatusMethodNotAllowed)

		return
	}

	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, "invalid_request", "malformed form body", http.StatusBadRequest)

		return
	}

	grantType := r.FormValue("grant_type")

	switch grantType {
	case grantTypeAuthorizationCode:
		h.handleAuthorizationCodeGrant(w, r)
	case grantTypeRefreshToken:
		h.handleRefreshTokenGrant(w, r)
	default:
		writeOAuthError(w, "unsupported_grant_type",
			"grant_type must be 'authorization_code' or 'refresh_token'",
			http.StatusBadRequest)
	}
}

func (h *Handler) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	authCode, valErr := h.validateAuthCodeGrant(r)
	if valErr != nil {
		writeOAuthError(w, valErr.code, valErr.description, valErr.status)

		return
	}

	client, err := h.store.GetClient(authCode.ClientID)
	if err != nil {
		writeOAuthError(w, "invalid_grant", "unknown client", http.StatusBadRequest)

		return
	}

	if !slices.Contains(client.GrantTypes, grantTypeAuthorizationCode) {
		writeOAuthError(w, "unauthorized_client",
			"client is not registered for authorization_code grant", http.StatusBadRequest)

		return
	}

	h.issueTokenPair(w, authCode.GitHubUsername, authCode.ClientID, authCode.Scopes, client.GrantTypes)
}

func (h *Handler) validateAuthCodeGrant(r *http.Request) (*Code, *oauthValidationError) {
	code := r.FormValue("code")
	if code == "" {
		return nil, &oauthValidationError{
			"invalid_request", "code is required", http.StatusBadRequest,
		}
	}

	codeVerifier := r.FormValue("code_verifier")
	if codeVerifier == "" {
		return nil, &oauthValidationError{
			"invalid_request", "code_verifier is required (PKCE mandatory)", http.StatusBadRequest,
		}
	}

	clientID := r.FormValue("client_id")
	if clientID == "" {
		return nil, &oauthValidationError{
			"invalid_request", "client_id is required", http.StatusBadRequest,
		}
	}

	redirectURI := r.FormValue("redirect_uri")
	if redirectURI == "" {
		return nil, &oauthValidationError{
			"invalid_request", "redirect_uri is required", http.StatusBadRequest,
		}
	}

	authCode, err := h.store.ConsumeAuthCode(code, time.Now())
	if err != nil {
		return nil, &oauthValidationError{
			"invalid_grant", "authorization code is invalid or expired", http.StatusBadRequest,
		}
	}

	if !verifyCodeChallenge(authCode.CodeChallenge, codeVerifier) {
		return nil, &oauthValidationError{
			"invalid_grant", "code_verifier does not match code_challenge", http.StatusBadRequest,
		}
	}

	if clientID != authCode.ClientID {
		return nil, &oauthValidationError{
			"invalid_grant", "client_id mismatch", http.StatusBadRequest,
		}
	}

	if redirectURI != authCode.RedirectURI {
		return nil, &oauthValidationError{
			"invalid_grant", "redirect_uri mismatch", http.StatusBadRequest,
		}
	}

	return authCode, nil
}

func (h *Handler) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	tokenValue := r.FormValue("refresh_token")
	if tokenValue == "" {
		writeOAuthError(w, "invalid_request", "refresh_token is required", http.StatusBadRequest)

		return
	}

	clientID := r.FormValue("client_id")
	if clientID == "" {
		writeOAuthError(w, "invalid_request", "client_id is required", http.StatusBadRequest)

		return
	}

	client, err := h.store.GetClient(clientID)
	if err != nil {
		writeOAuthError(w, "invalid_grant", "unknown client", http.StatusBadRequest)

		return
	}

	if !slices.Contains(client.GrantTypes, grantTypeRefreshToken) {
		writeOAuthError(w, "unauthorized_client",
			"client is not registered for refresh_token grant", http.StatusBadRequest)

		return
	}

	now := time.Now()

	refreshTok, err := h.store.ConsumeRefreshToken(tokenValue, now)
	if err != nil {
		writeOAuthError(w, "invalid_grant", "refresh token is invalid or expired", http.StatusBadRequest)

		return
	}

	if clientID != refreshTok.ClientID {
		writeOAuthError(w, "invalid_grant", "refresh token is invalid or expired", http.StatusBadRequest)

		return
	}

	h.issueTokenPair(w, refreshTok.GitHubUsername, refreshTok.ClientID, refreshTok.Scopes, client.GrantTypes)
}

func (h *Handler) issueTokenPair(
	w http.ResponseWriter, username, clientID string, scopes, grantTypes []string,
) {
	accessToken, err := IssueAccessToken(h.jwtSecret, h.issuer, accessTokenTTL, username, scopes)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	var refreshToken string

	if slices.Contains(grantTypes, grantTypeRefreshToken) {
		refreshToken, err = IssueRefreshToken()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		h.store.SaveRefreshToken(&RefreshToken{
			Token:         refreshToken,
			ClientID:      clientID,
			GitHubUsername: username,
			Scopes:        scopes,
			ExpiresAt:     time.Now().Add(refreshTokenTTL),
		})
	}

	writeTokenResponse(w, accessToken, refreshToken, int(accessTokenTTL.Seconds()))
}

// registrationRequest represents the JSON body of a dynamic client registration request (RFC 7591).
type registrationRequest struct {
	RedirectURIs []string `json:"redirect_uris"`
	ClientName   string   `json:"client_name,omitempty"`
	GrantTypes   []string `json:"grant_types,omitempty"`
}

// registrationResponse represents the JSON response for a successful client registration.
type registrationResponse struct {
	ClientID     string   `json:"client_id"`
	RedirectURIs []string `json:"redirect_uris"`
	ClientName   string   `json:"client_name,omitempty"`
	GrantTypes   []string `json:"grant_types"`
}

// HandleRegister handles POST /oauth/register for dynamic client registration per RFC 7591.
func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOAuthError(w, "invalid_request", "method must be POST", http.StatusMethodNotAllowed)

		return
	}

	var req registrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOAuthError(w, "invalid_client_metadata", "malformed JSON body", http.StatusBadRequest)

		return
	}

	if err := validateRedirectURIs(req.RedirectURIs); err != "" {
		writeOAuthError(w, "invalid_client_metadata", err, http.StatusBadRequest)

		return
	}

	grantTypes, grantErr := validateGrantTypes(req.GrantTypes)
	if grantErr != "" {
		writeOAuthError(w, "invalid_client_metadata", grantErr, http.StatusBadRequest)

		return
	}

	clientID, err := uuid.NewV4()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	client := &RegisteredClient{
		ClientID:     clientID.String(),
		RedirectURIs: req.RedirectURIs,
		ClientName:   req.ClientName,
		GrantTypes:   grantTypes,
		CreatedAt:    time.Now(),
	}

	if err = h.store.SaveClient(client); err != nil {
		writeOAuthError(w, "server_error", "too many registered clients", http.StatusServiceUnavailable)

		return
	}

	resp := registrationResponse{
		ClientID:     client.ClientID,
		RedirectURIs: client.RedirectURIs,
		ClientName:   client.ClientName,
		GrantTypes:   client.GrantTypes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode registration response", "error", err)
	}
}

// validateRedirectURIs checks that redirect URIs are present, use http/https scheme,
// have a non-empty host, and contain no fragment (per OAuth 2.1 Section 2.3.1).
// HTTP scheme is only allowed for loopback addresses (127.0.0.1, [::1], localhost).
func validateRedirectURIs(uris []string) string {
	if len(uris) == 0 {
		return "redirect_uris is required"
	}

	for _, uri := range uris {
		parsed, err := url.Parse(uri)
		if err != nil {
			return "invalid redirect_uri: " + uri
		}

		if parsed.Scheme != schemeHTTP && parsed.Scheme != schemeHTTPS {
			return "redirect_uri must use http or https scheme: " + uri
		}

		if parsed.Host == "" {
			return "redirect_uri must have a host: " + uri
		}

		if parsed.Fragment != "" {
			return "redirect_uri must not contain a fragment: " + uri
		}

		if parsed.Scheme == schemeHTTP && !isLoopbackHost(parsed.Hostname()) {
			return "redirect_uri with http scheme is only allowed for loopback addresses: " + uri
		}
	}

	return ""
}

// isLoopbackHost returns true if the hostname is a loopback address.
func isLoopbackHost(hostname string) bool {
	return hostname == "127.0.0.1" || hostname == "::1" || hostname == "localhost"
}

// validateGrantTypes validates and defaults the grant_types list.
// Returns the validated list and an error string (empty if valid).
func validateGrantTypes(grantTypes []string) ([]string, string) {
	if len(grantTypes) == 0 {
		return []string{grantTypeAuthorizationCode}, ""
	}

	for _, gt := range grantTypes {
		if gt != grantTypeAuthorizationCode && gt != grantTypeRefreshToken {
			return nil, "unsupported grant_type: " + gt + "; only authorization_code and refresh_token are supported"
		}
	}

	return grantTypes, ""
}

// signState creates an HMAC-SHA256 signed state string from the authorize parameters.
// Format: base64url(json) + "." + base64url(hmac).
func (h *Handler) signState(authState authorizeState) (string, error) {
	data, err := json.Marshal(authState)
	if err != nil {
		return "", fmt.Errorf("marshaling state: %w", err)
	}

	payload := base64.RawURLEncoding.EncodeToString(data)

	mac := hmac.New(sha256.New, []byte(h.jwtSecret))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return payload + "." + sig, nil
}

// verifyState validates the HMAC signature and unmarshals the authorize state.
func (h *Handler) verifyState(signed string) (authorizeState, error) {
	parts := strings.SplitN(signed, ".", signedStateParts)

	if len(parts) != signedStateParts {
		return authorizeState{}, errInvalidState
	}

	payload := parts[0]
	sigEncoded := parts[1]

	mac := hmac.New(sha256.New, []byte(h.jwtSecret))
	mac.Write([]byte(payload))
	expectedSig := mac.Sum(nil)

	actualSig, err := base64.RawURLEncoding.DecodeString(sigEncoded)
	if err != nil {
		return authorizeState{}, fmt.Errorf("%w: %w", errInvalidState, err)
	}

	if !hmac.Equal(expectedSig, actualSig) {
		return authorizeState{}, errInvalidState
	}

	data, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return authorizeState{}, fmt.Errorf("%w: %w", errInvalidState, err)
	}

	var authState authorizeState
	if err := json.Unmarshal(data, &authState); err != nil {
		return authorizeState{}, fmt.Errorf("%w: %w", errInvalidState, err)
	}

	if time.Since(time.Unix(authState.CreatedAt, 0)) > stateTTL {
		return authorizeState{}, errInvalidState
	}

	return authState, nil
}

// verifyCodeChallenge performs S256 PKCE verification using constant-time comparison.
// challenge = BASE64URL(SHA256(verifier)).
func verifyCodeChallenge(challenge, verifier string) bool {
	digest := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(digest[:])

	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// tokenResponse is the JSON response for successful token requests.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func writeTokenResponse(
	w http.ResponseWriter, accessToken, refreshToken string, expiresIn int,
) {
	resp := tokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		RefreshToken: refreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode token response", "error", err)
	}
}

// oauthErrorResponse is the JSON error response per RFC 6749 Section 5.2.
type oauthErrorResponse struct {
	Error       string `json:"error"`
	Description string `json:"error_description,omitempty"`
}

func writeOAuthError(w http.ResponseWriter, errCode, description string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(oauthErrorResponse{
		Error:       errCode,
		Description: description,
	}); err != nil {
		slog.Error("failed to encode OAuth error response", "error", err)
	}
}

func generateRandomString(length int) (string, error) {
	buf := make([]byte, length)

	_, err := rand.Read(buf)
	if err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func parseScopes(scope string) []string {
	if strings.TrimSpace(scope) == "" {
		return nil
	}

	return strings.Fields(scope)
}
