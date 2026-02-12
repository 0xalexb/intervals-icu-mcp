package client

import "net/http"

// NewTestClient creates a Client with a custom base URL and HTTP client for testing.
func NewTestClient(baseURL, apiKey, athleteID string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		athleteID:  athleteID,
		httpClient: httpClient,
	}
}
