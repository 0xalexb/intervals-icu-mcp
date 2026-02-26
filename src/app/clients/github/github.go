// Package github provides an HTTP client for the GitHub OAuth and user API endpoints.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTokenURL       = "https://github.com/login/oauth/access_token"
	defaultUserURL        = "https://api.github.com/user"
	httpTimeout           = 10 * time.Second
	maxGitHubResponseSize = 1 << 20 // 1 MB
	userAgent             = "intervals-icu-mcp"
)

// ErrGitHubTokenExchange is returned when the GitHub OAuth token exchange fails.
var ErrGitHubTokenExchange = errors.New("GitHub token exchange failed")

// ErrGitHubUserFetch is returned when fetching the GitHub user profile fails.
var ErrGitHubUserFetch = errors.New("GitHub user fetch failed")

// User represents the authenticated GitHub user's profile.
type User struct {
	Login string `json:"login"`
}

// Client performs HTTP calls against the GitHub OAuth and API endpoints.
type Client struct {
	tokenURL   string
	userURL    string
	httpClient *http.Client
}

// NewClient creates a Client pointing at the real GitHub endpoints.
func NewClient() *Client {
	return &Client{
		tokenURL:   defaultTokenURL,
		userURL:    defaultUserURL,
		httpClient: &http.Client{Timeout: httpTimeout},
	}
}

// ExchangeCode exchanges a GitHub authorization code for an access token
// by POSTing to GitHub's OAuth token endpoint.
func (c *Client) ExchangeCode(
	ctx context.Context,
	clientID string,
	clientSecret string,
	code string,
) (string, error) {
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrGitHubTokenExchange, err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxGitHubResponseSize))
	if err != nil {
		return "", fmt.Errorf("%w: reading response: %w", ErrGitHubTokenExchange, err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", ErrGitHubTokenExchange, resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("%w: decoding response: %w", ErrGitHubTokenExchange, err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("%w: %s: %s", ErrGitHubTokenExchange, tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("%w: empty access token in response", ErrGitHubTokenExchange)
	}

	return tokenResp.AccessToken, nil
}

// GetUser fetches the authenticated GitHub user's profile using the
// provided access token.
func (c *Client) GetUser(ctx context.Context, accessToken string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating user request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubUserFetch, err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxGitHubResponseSize))
	if err != nil {
		return nil, fmt.Errorf("%w: reading response: %w", ErrGitHubUserFetch, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrGitHubUserFetch, resp.StatusCode)
	}

	var user User

	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("%w: decoding response: %w", ErrGitHubUserFetch, err)
	}

	if user.Login == "" {
		return nil, fmt.Errorf("%w: empty login in response", ErrGitHubUserFetch)
	}

	return &user, nil
}
