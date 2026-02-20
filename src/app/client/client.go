// Package client provides an HTTP client for the Intervals.icu API.
package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	baseURL            = "https://intervals.icu"
	httpClientTimeout  = 30 * time.Second
	maxResponseSize = 10 << 20 // 10 MB
)

var (
	errMissingAPIKey     = errors.New("INTERVALS_API_KEY is required")
	errMissingAthleteID  = errors.New("INTERVALS_ATHLETE_ID is required")
	errRequestFailed     = errors.New("API request failed")
	errResponseTruncated = errors.New("API response exceeded maximum size")
)

// Config holds the configuration for the Intervals.icu API client.
type Config struct {
	APIKey    string `json:"-"`
	AthleteID string `json:"-"`
}

// Client is an HTTP client for the Intervals.icu API.
type Client struct {
	baseURL    string
	apiKey     string
	athleteID  string
	httpClient *http.Client
}

// NewClient creates a new Intervals.icu API client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, errMissingAPIKey
	}

	if cfg.AthleteID == "" {
		return nil, errMissingAthleteID
	}

	return &Client{
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		athleteID:  cfg.AthleteID,
		httpClient: &http.Client{Timeout: httpClientTimeout},
	}, nil
}

// AthleteID returns the configured athlete ID.
func (c *Client) AthleteID() string {
	return c.athleteID
}

// Get sends a GET request to the given path with optional query parameters.
func (c *Client) Get(ctx context.Context, path string, queryParams url.Values) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, queryParams, http.NoBody)
}

// Post sends a POST request to the given path with the provided body.
func (c *Client) Post(ctx context.Context, path string, body io.Reader) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, path, nil, body)
}

// Put sends a PUT request to the given path with the provided body.
func (c *Client) Put(ctx context.Context, path string, body io.Reader) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPut, path, nil, body)
}

// Delete sends a DELETE request to the given path with optional query parameters.
func (c *Client) Delete(ctx context.Context, path string, queryParams url.Values) ([]byte, error) {
	return c.doRequest(ctx, http.MethodDelete, path, queryParams, http.NoBody)
}

func (c *Client) doRequest(
	ctx context.Context,
	method string,
	path string,
	queryParams url.Values,
	body io.Reader,
) ([]byte, error) {
	fullURL := c.baseURL + path
	if len(queryParams) > 0 {
		fullURL += "?" + queryParams.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.SetBasicAuth("API_KEY", c.apiKey)
	req.Header.Set("Accept", "application/json")

	if body != http.NoBody {
		req.Header.Set("Content-Type", "application/json")
	}

	//nolint:gosec // G107: baseURL is constant; paths are from internal tool code.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	closeErr := resp.Body.Close()

	if readErr != nil {
		return nil, errors.Join(fmt.Errorf("reading response body: %w", readErr), closeErr)
	}

	if int64(len(respBody)) > maxResponseSize {
		return nil, errors.Join(fmt.Errorf("%w: %d bytes", errResponseTruncated, maxResponseSize), closeErr)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, errors.Join(fmt.Errorf("%w: status %d", errRequestFailed, resp.StatusCode), closeErr)
	}

	if closeErr != nil {
		return respBody, fmt.Errorf("closing response body: %w", closeErr)
	}

	return respBody, nil
}
