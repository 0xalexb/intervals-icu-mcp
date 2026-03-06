// Package rest provides OAuth 2.1 HTTP handlers for the MCP server.
package rest

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"go.uber.org/fx"
	"golang.org/x/crypto/hkdf"

	"github.com/0xalexb/intervals-icu-mcp/src/app/auth"
	"github.com/0xalexb/intervals-icu-mcp/src/app/clients/github"
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

	scopeMCP = "mcp"

	genericErrorDescription = "an error occurred during authorization"

	maxRedirectURIs    = 10
	maxClientNameBytes = 256
)

var (
	errInvalidState              = errors.New("invalid or tampered state parameter")
	errRedirectURIsRequired      = errors.New("redirect_uris is required")
	errInvalidRedirectURI        = errors.New("invalid redirect_uri")
	errRedirectURIBadScheme      = errors.New("redirect_uri must use http or https scheme")
	errRedirectURIMissingHost    = errors.New("redirect_uri must have a host")
	errRedirectURIHasFragment    = errors.New("redirect_uri must not contain a fragment")
	errRedirectURINotLoopback    = errors.New("redirect_uri with http scheme is only allowed for loopback addresses")
	errTooManyRedirectURIs       = errors.New("too many redirect_uris")
	errClientNameTooLong         = errors.New("client_name too long")
	errUnsupportedGrantType = errors.New("unsupported grant_type")
)

// HandlerParams holds the DI-injected dependencies for the OAuth Handler.
type HandlerParams struct {
	fx.In

	Store                       *auth.Store
	AllowedUsers                auth.AllowedUsers
	GitHubClientID              auth.GitHubClientID
	GitHubClientSecret          auth.GitHubClientSecret
	JWTSecret                   auth.JWTSecret
	Issuer                      auth.Issuer
	AuthorizationServerMetadata *auth.AuthorizationServerMetadata
	GitHubClient                *github.Client
}

// Handler implements the OAuth 2.1 HTTP handlers.
type Handler struct {
	store          *auth.Store
	allowedUsers   auth.AllowedUsers
	ghClientID     auth.GitHubClientID
	ghClientSecret auth.GitHubClientSecret
	jwtSecret      auth.JWTSecret
	stateKey       []byte
	issuer         auth.Issuer
	metadata       *auth.AuthorizationServerMetadata
	ghClient       *github.Client
}

// NewHandler creates an OAuth Handler from DI-injected dependencies.
func NewHandler(p HandlerParams) (*Handler, error) {
	stateKey, err := deriveStateKey([]byte(p.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("deriving state HMAC key: %w", err)
	}

	return &Handler{
		store:          p.Store,
		allowedUsers:   p.AllowedUsers,
		ghClientID:     p.GitHubClientID,
		ghClientSecret: p.GitHubClientSecret,
		jwtSecret:      p.JWTSecret,
		stateKey:       stateKey,
		issuer:         p.Issuer,
		metadata:       p.AuthorizationServerMetadata,
		ghClient:       p.GitHubClient,
	}, nil
}

// deriveStateKey derives a separate HMAC key for OAuth state signing using HKDF.
func deriveStateKey(secret []byte) ([]byte, error) {
	reader := hkdf.New(sha256.New, secret, nil, []byte("oauth-state-hmac"))

	key := make([]byte, sha256.Size)

	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, fmt.Errorf("reading HKDF output: %w", err)
	}

	return key, nil
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

	client, err := h.store.GetClient(authState.ClientID, time.Now())
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
		h.redirectError(w, r, authState, "server_error", "internal error")

		return
	}

	ghURL, err := url.Parse(githubAuthorizeURL)
	if err != nil {
		h.redirectError(w, r, authState, "server_error", "internal error")

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
				"invalid_scope", "only 'mcp' scope is supported", http.StatusBadRequest,
			}
		}
	}

	return scope, nil
}

// HandleCallback handles GET /oauth/callback. It validates the HMAC-signed state,
// exchanges the GitHub authorization code for an access token, fetches the GitHub user,
// checks the allowlist, generates an auth code, and redirects back to the client.
func (h *Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	code, authState, valErr := h.validateCallbackParams(query)
	if valErr != nil {
		h.redirectOrWriteError(w, r, query.Get("state"), valErr)

		return
	}

	ghUser, ghErr := h.resolveGitHubUser(r, code)
	if ghErr != nil {
		h.redirectError(w, r, authState, ghErr.code, ghErr.description)

		return
	}

	authCode, err := generateRandomString(authCodeLength)
	if err != nil {
		h.redirectError(w, r, authState, "server_error", "internal error")

		return
	}

	if err = h.store.SaveAuthCode(&auth.Code{
		Code:                authCode,
		ClientID:            authState.ClientID,
		RedirectURI:         authState.RedirectURI,
		CodeChallenge:       authState.CodeChallenge,
		CodeChallengeMethod: authState.CodeChallengeMethod,
		GitHubUsername:      strings.ToLower(ghUser.Login),
		Scopes:              parseScopes(authState.Scope),
		ExpiresAt:           time.Now().Add(authCodeTTL),
	}); err != nil {
		h.redirectError(w, r, authState, "server_error", "too many pending authorization codes")

		return
	}

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

// redirectOrWriteError attempts to redirect the OAuth error back to the client's redirect_uri
// by verifying the HMAC-signed state. If the state is absent or invalid, it falls back to a
// JSON error response (the best we can do when we can't identify the client).
func (h *Handler) redirectOrWriteError(
	w http.ResponseWriter, r *http.Request, signedState string, valErr *oauthValidationError,
) {
	if signedState == "" {
		writeOAuthError(w, valErr.code, valErr.description, valErr.status)

		return
	}

	authState, err := h.verifyState(signedState)
	if err != nil {
		writeOAuthError(w, valErr.code, valErr.description, valErr.status)

		return
	}

	h.redirectError(w, r, authState, valErr.code, valErr.description)
}

// redirectError sends an OAuth error back to the client's redirect_uri as query parameters
// per RFC 6749 Section 4.1.2.1.
func (h *Handler) redirectError(
	w http.ResponseWriter, r *http.Request, authState authorizeState, errCode, description string,
) {
	redirectURL, err := url.Parse(authState.RedirectURI)
	if err != nil {
		writeOAuthError(w, errCode, description, http.StatusBadRequest)

		return
	}

	redirectQuery := redirectURL.Query()
	redirectQuery.Set("error", errCode)
	redirectQuery.Set("error_description", description)

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
		code, desc := sanitizeGitHubError(errParam, query.Get("error_description"))

		return "", authorizeState{}, &oauthValidationError{
			code, desc, http.StatusBadRequest,
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
) (*github.User, *oauthValidationError) {
	ghToken, err := h.ghClient.ExchangeCode(r.Context(), string(h.ghClientID), string(h.ghClientSecret), code)
	if err != nil {
		slog.Error("GitHub code exchange failed", "error", err)

		return nil, &oauthValidationError{
			"server_error", "failed to exchange GitHub code", http.StatusInternalServerError,
		}
	}

	ghUser, err := h.ghClient.GetUser(r.Context(), ghToken)
	if err != nil {
		slog.Error("GitHub user fetch failed", "error", err)

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

	client, err := h.store.GetClient(authCode.ClientID, time.Now())
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

func (h *Handler) validateAuthCodeGrant(r *http.Request) (*auth.Code, *oauthValidationError) {
	p, valErr := extractAuthCodeParams(r)
	if valErr != nil {
		return nil, valErr
	}

	authCode, err := h.store.ValidateAndConsumeAuthCode(p.code, time.Now(), func(ac *auth.Code) error {
		if !verifyCodeChallenge(ac.CodeChallenge, p.codeVerifier) {
			return &oauthValidationError{
				"invalid_grant", "code_verifier does not match code_challenge", http.StatusBadRequest,
			}
		}

		if p.clientID != ac.ClientID {
			return &oauthValidationError{
				"invalid_grant", "client_id mismatch", http.StatusBadRequest,
			}
		}

		if p.redirectURI != ac.RedirectURI {
			return &oauthValidationError{
				"invalid_grant", "redirect_uri mismatch", http.StatusBadRequest,
			}
		}

		return nil
	})
	if err != nil {
		var valErr *oauthValidationError
		if errors.As(err, &valErr) {
			return nil, valErr
		}

		return nil, &oauthValidationError{
			"invalid_grant", "authorization code is invalid or expired", http.StatusBadRequest,
		}
	}

	return authCode, nil
}

// authCodeParams holds the validated form parameters for an authorization code grant.
type authCodeParams struct {
	code, codeVerifier, clientID, redirectURI string
}

func extractAuthCodeParams(r *http.Request) (authCodeParams, *oauthValidationError) {
	p := authCodeParams{
		code:         r.FormValue("code"),
		codeVerifier: r.FormValue("code_verifier"),
		clientID:     r.FormValue("client_id"),
		redirectURI:  r.FormValue("redirect_uri"),
	}

	switch {
	case p.code == "":
		return p, &oauthValidationError{"invalid_request", "code is required", http.StatusBadRequest}
	case p.codeVerifier == "":
		return p, &oauthValidationError{
			"invalid_request", "code_verifier is required (PKCE mandatory)", http.StatusBadRequest,
		}
	case p.clientID == "":
		return p, &oauthValidationError{"invalid_request", "client_id is required", http.StatusBadRequest}
	case p.redirectURI == "":
		return p, &oauthValidationError{"invalid_request", "redirect_uri is required", http.StatusBadRequest}
	}

	return p, nil
}

func (h *Handler) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	tokenValue, clientID, valErr := h.validateRefreshTokenGrant(r)
	if valErr != nil {
		writeOAuthError(w, valErr.code, valErr.description, valErr.status)

		return
	}

	now := time.Now()

	newRefreshTokenValue, err := auth.IssueRefreshToken()
	if err != nil {
		writeOAuthError(w, "server_error", "internal error", http.StatusInternalServerError)

		return
	}

	newRefreshToken := &auth.RefreshToken{
		Token:     newRefreshTokenValue,
		ExpiresAt: now.Add(refreshTokenTTL),
	}

	refreshTok, err := h.store.RotateRefreshToken(tokenValue, clientID, now, newRefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrMaxRefreshTokensReached) {
			writeOAuthError(w, "server_error", "too many active refresh tokens", http.StatusServiceUnavailable)

			return
		}

		writeOAuthError(w, "invalid_grant", "refresh token is invalid or expired", http.StatusBadRequest)

		return
	}

	accessToken, err := auth.IssueAccessToken(
		h.jwtSecret, h.issuer, accessTokenTTL, refreshTok.GitHubUsername, refreshTok.Scopes,
	)
	if err != nil {
		writeOAuthError(w, "server_error", "internal error", http.StatusInternalServerError)

		return
	}

	writeTokenResponse(w, accessToken, newRefreshTokenValue, int(accessTokenTTL.Seconds()))
}

func (h *Handler) validateRefreshTokenGrant(r *http.Request) (string, string, *oauthValidationError) {
	tokenValue := r.FormValue("refresh_token")
	if tokenValue == "" {
		return "", "", &oauthValidationError{
			"invalid_request", "refresh_token is required", http.StatusBadRequest,
		}
	}

	clientID := r.FormValue("client_id")
	if clientID == "" {
		return "", "", &oauthValidationError{
			"invalid_request", "client_id is required", http.StatusBadRequest,
		}
	}

	client, err := h.store.GetClient(clientID, time.Now())
	if err != nil {
		return "", "", &oauthValidationError{
			"invalid_grant", "unknown client", http.StatusBadRequest,
		}
	}

	if !slices.Contains(client.GrantTypes, grantTypeRefreshToken) {
		return "", "", &oauthValidationError{
			"unauthorized_client", "client is not registered for refresh_token grant", http.StatusBadRequest,
		}
	}

	return tokenValue, clientID, nil
}

func (h *Handler) issueTokenPair(
	w http.ResponseWriter, username, clientID string, scopes, grantTypes []string,
) {
	accessToken, err := auth.IssueAccessToken(h.jwtSecret, h.issuer, accessTokenTTL, username, scopes)
	if err != nil {
		writeOAuthError(w, "server_error", "internal error", http.StatusInternalServerError)

		return
	}

	var refreshToken string

	if slices.Contains(grantTypes, grantTypeRefreshToken) {
		refreshToken, err = auth.IssueRefreshToken()
		if err != nil {
			writeOAuthError(w, "server_error", "internal error", http.StatusInternalServerError)

			return
		}

		if err = h.store.SaveRefreshToken(&auth.RefreshToken{
			Token:         refreshToken,
			ClientID:      clientID,
			GitHubUsername: username,
			Scopes:        scopes,
			ExpiresAt:     time.Now().Add(refreshTokenTTL),
		}); err != nil {
			writeOAuthError(w, "server_error", "too many active refresh tokens", http.StatusServiceUnavailable)

			return
		}
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
	var req registrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOAuthError(w, "invalid_client_metadata", "malformed JSON body", http.StatusBadRequest)

		return
	}

	grantTypes, valErr := validateRegistrationRequest(&req)
	if valErr != nil {
		writeOAuthError(w, "invalid_client_metadata", valErr.Error(), http.StatusBadRequest)

		return
	}

	clientID, err := uuid.NewV4()
	if err != nil {
		writeOAuthError(w, "server_error", "internal error", http.StatusInternalServerError)

		return
	}

	client := &auth.RegisteredClient{
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

const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

// validateRegistrationRequest validates client_name length, redirect URIs, and grant types.
func validateRegistrationRequest(req *registrationRequest) ([]string, error) {
	if len(req.ClientName) > maxClientNameBytes {
		return nil, errClientNameTooLong
	}

	if err := validateRedirectURIs(req.RedirectURIs); err != nil {
		return nil, err
	}

	return validateGrantTypes(req.GrantTypes)
}

// validateRedirectURIs checks that redirect URIs are present and within limits,
// then delegates per-URI validation to validateRedirectURI.
func validateRedirectURIs(uris []string) error {
	if len(uris) == 0 {
		return errRedirectURIsRequired
	}

	if len(uris) > maxRedirectURIs {
		return fmt.Errorf("%w: maximum %d allowed", errTooManyRedirectURIs, maxRedirectURIs)
	}

	for _, uri := range uris {
		if err := validateRedirectURI(uri); err != nil {
			return err
		}
	}

	return nil
}

// validateRedirectURI checks a single redirect URI uses http/https scheme,
// has a non-empty host, and contains no fragment (per OAuth 2.1 Section 2.3.1).
// HTTP scheme is only allowed for loopback addresses (127.0.0.1, [::1], localhost).
func validateRedirectURI(uri string) error {
	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidRedirectURI, uri)
	}

	if parsed.Scheme != schemeHTTP && parsed.Scheme != schemeHTTPS {
		return fmt.Errorf("%w: %s", errRedirectURIBadScheme, uri)
	}

	if parsed.Hostname() == "" {
		return fmt.Errorf("%w: %s", errRedirectURIMissingHost, uri)
	}

	if parsed.Fragment != "" {
		return fmt.Errorf("%w: %s", errRedirectURIHasFragment, uri)
	}

	if parsed.Scheme == schemeHTTP && !isLoopbackHost(parsed.Hostname()) {
		return fmt.Errorf("%w: %s", errRedirectURINotLoopback, uri)
	}

	return nil
}

// isLoopbackHost returns true if the hostname is a loopback address.
// It covers the full 127.0.0.0/8 range and IPv6 ::1 via net.IP.IsLoopback,
// plus "localhost" as a hostname.
func isLoopbackHost(hostname string) bool {
	if hostname == "localhost" {
		return true
	}

	ip := net.ParseIP(hostname)

	return ip != nil && ip.IsLoopback()
}

// validateGrantTypes validates and defaults the grant_types list.
func validateGrantTypes(grantTypes []string) ([]string, error) {
	if len(grantTypes) == 0 {
		return []string{grantTypeAuthorizationCode}, nil
	}

	for _, gt := range grantTypes {
		if gt != grantTypeAuthorizationCode && gt != grantTypeRefreshToken {
			return nil, fmt.Errorf("%w: %s", errUnsupportedGrantType, gt)
		}
	}

	return grantTypes, nil
}

// signState creates an HMAC-SHA256 signed state string from the authorize parameters.
// Format: base64url(json) + "." + base64url(hmac).
func (h *Handler) signState(authState authorizeState) (string, error) {
	data, err := json.Marshal(authState)
	if err != nil {
		return "", fmt.Errorf("marshaling state: %w", err)
	}

	payload := base64.RawURLEncoding.EncodeToString(data)

	mac := hmac.New(sha256.New, h.stateKey)
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

	mac := hmac.New(sha256.New, h.stateKey)
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

// sanitizeGitHubError replaces unrecognized OAuth error codes and descriptions
// returned by GitHub with safe defaults to prevent content injection.
func sanitizeGitHubError(errCode, description string) (string, string) {
	if !isKnownOAuthError(errCode) {
		return "server_error", genericErrorDescription
	}

	if !isCleanDescription(description) {
		return errCode, genericErrorDescription
	}

	return errCode, description
}

// isCleanDescription checks that the description contains only safe characters
// (printable ASCII without angle brackets or other HTML/script-injection vectors).
func isCleanDescription(s string) bool {
	if s == "" || len(s) > 256 {
		return false
	}

	for _, c := range s {
		if c < ' ' || c > '~' || c == '<' || c == '>' {
			return false
		}
	}

	return true
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

// isKnownOAuthError returns true if the error code is a recognized OAuth error code
// that GitHub may return during the authorization flow.
func isKnownOAuthError(code string) bool {
	switch code {
	case "access_denied", "temporarily_unavailable", "server_error", "invalid_request",
		"unauthorized_client", "unsupported_response_type", "invalid_scope", "interaction_required",
		"application_suspended", "redirect_uri_mismatch":
		return true
	default:
		return false
	}
}
