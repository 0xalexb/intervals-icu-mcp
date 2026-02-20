package api

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	errOriginMissingScheme  = errors.New("must be a full origin URL with scheme (e.g., http://localhost:3000)")
	errOriginHasPath        = errors.New("must be an origin URL without path")
	errOriginTrailingSlash  = errors.New("must not end with a trailing slash")
	errOriginIsWildcard     = errors.New("wildcard '*' is not supported, specify explicit origins")
	errOriginHasQuery       = errors.New("must be an origin URL without query parameters")
	errOriginHasFragment    = errors.New("must be an origin URL without fragment")
	errOriginEmptyHost      = errors.New("must have a non-empty host")
)

// RawAllowedOrigins is the raw comma-separated flag value for allowed CORS origins,
// injected from main.go via DI.
type RawAllowedOrigins string

// AllowedOrigins is the parsed and validated list of full origin URLs allowed for CORS.
type AllowedOrigins []string

// NewAllowedOrigins parses and validates a RawAllowedOrigins value into an AllowedOrigins list.
// It splits by comma, trims whitespace, drops empty entries, and validates each entry is a
// full origin URL with scheme and host (e.g., http://localhost:3000).
func NewAllowedOrigins(raw RawAllowedOrigins) (AllowedOrigins, error) {
	if strings.TrimSpace(string(raw)) == "" {
		return AllowedOrigins{}, nil
	}

	parts := strings.Split(string(raw), ",")
	origins := make(AllowedOrigins, 0, len(parts))

	for _, part := range parts {
		entry := strings.TrimSpace(part)
		if entry == "" {
			continue
		}

		err := validateOrigin(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed origin %q: %w", entry, err)
		}

		origins = append(origins, entry)
	}

	return origins, nil
}

// Hostnames extracts bare hostnames from each full origin URL.
// For example, "http://localhost:3000" yields "localhost" and "http://[::1]:9090" yields "::1".
func (o AllowedOrigins) Hostnames() []string {
	hostnames := make([]string, 0, len(o))

	for _, entry := range o {
		parsed, err := url.Parse(entry)
		if err != nil {
			continue
		}

		hostnames = append(hostnames, parsed.Hostname())
	}

	return hostnames
}

// validateOrigin checks that the entry is a full origin URL with scheme and host.
// It requires http:// or https:// scheme, a non-empty host, and no path/query/fragment.
func validateOrigin(entry string) error {
	if entry == "*" {
		return errOriginIsWildcard
	}

	parsed, err := url.Parse(entry)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errOriginMissingScheme
	}

	if parsed.Host == "" {
		return errOriginEmptyHost
	}

	if parsed.Path == "/" {
		return errOriginTrailingSlash
	}

	if parsed.Path != "" {
		return errOriginHasPath
	}

	if parsed.RawQuery != "" {
		return errOriginHasQuery
	}

	if parsed.Fragment != "" {
		return errOriginHasFragment
	}

	return nil
}
