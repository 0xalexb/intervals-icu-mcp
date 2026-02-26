package github

import "net/http"

// NewTestClient creates a Client with custom token and user endpoint URLs for testing.
func NewTestClient(tokenURL, userURL string) *Client {
	return &Client{
		tokenURL:   tokenURL,
		userURL:    userURL,
		httpClient: http.DefaultClient,
	}
}
