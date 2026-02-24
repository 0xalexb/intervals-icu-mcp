package auth

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
	githubTokenURL          = "https://github.com/login/oauth/access_token"
	githubUserURL           = "https://api.github.com/user"
	githubTimeout           = 10 * time.Second
	maxGitHubResponseSize   = 1 << 20 // 1 MB
)

var (
	errGitHubTokenExchange = errors.New("GitHub token exchange failed")
	errGitHubUserFetch     = errors.New("GitHub user fetch failed")
)

// GitHubUser represents the authenticated GitHub user's profile.
type GitHubUser struct {
	Login string `json:"login"`
}

// GitHubClient performs HTTP calls against the GitHub OAuth and API endpoints.
type GitHubClient struct {
	tokenURL   string
	userURL    string
	httpClient *http.Client
}

// NewGitHubClient creates a GitHubClient pointing at the real GitHub endpoints.
func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		tokenURL:   githubTokenURL,
		userURL:    githubUserURL,
		httpClient: &http.Client{Timeout: githubTimeout},
	}
}

// ExchangeGitHubCode exchanges a GitHub authorization code for an access token
// by POSTing to GitHub's OAuth token endpoint.
func (g *GitHubClient) ExchangeGitHubCode(
	ctx context.Context,
	clientID GitHubClientID,
	clientSecret GitHubClientSecret,
	code string,
) (string, error) {
	form := url.Values{
		"client_id":     {string(clientID)},
		"client_secret": {string(clientSecret)},
		"code":          {code},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errGitHubTokenExchange, err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxGitHubResponseSize))
	if err != nil {
		return "", fmt.Errorf("%w: reading response: %w", errGitHubTokenExchange, err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", errGitHubTokenExchange, resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("%w: decoding response: %w", errGitHubTokenExchange, err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("%w: %s: %s", errGitHubTokenExchange, tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("%w: empty access token in response", errGitHubTokenExchange)
	}

	return tokenResp.AccessToken, nil
}

// GetGitHubUser fetches the authenticated GitHub user's profile using the
// provided access token.
func (g *GitHubClient) GetGitHubUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.userURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating user request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errGitHubUserFetch, err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxGitHubResponseSize))
	if err != nil {
		return nil, fmt.Errorf("%w: reading response: %w", errGitHubUserFetch, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", errGitHubUserFetch, resp.StatusCode)
	}

	var user GitHubUser

	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("%w: decoding response: %w", errGitHubUserFetch, err)
	}

	if user.Login == "" {
		return nil, fmt.Errorf("%w: empty login in response", errGitHubUserFetch)
	}

	return &user, nil
}
