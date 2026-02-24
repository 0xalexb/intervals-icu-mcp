// Package auth provides OAuth 2.1 authentication for the MCP server's streamable HTTP transport.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

const jwtSecretLength = 32

var (
	errIssuerMissingScheme    = errors.New("issuer must have http or https scheme")
	errIssuerEmptyHost        = errors.New("issuer must have a non-empty host")
	errIssuerHasPath          = errors.New("issuer must not contain a path")
	errIssuerHasQuery         = errors.New("issuer must not contain a query string")
	errIssuerHasFragment      = errors.New("issuer must not contain a fragment")
	errJWTSecretTooShort      = errors.New("jwt secret too short")
)

// GitHubClientID is the GitHub OAuth app client ID, injected from main.go via DI.
type GitHubClientID string

// GitHubClientSecret is the GitHub OAuth app client secret, injected from main.go via DI.
type GitHubClientSecret string

// RawAllowedUsers is the raw comma-separated flag value for allowed GitHub usernames,
// injected from main.go via DI.
type RawAllowedUsers string

// RawJWTSecret is the raw CLI flag value for the JWT signing key, injected from main.go via DI.
type RawJWTSecret string

// JWTSecret is the validated HMAC-SHA256 signing key for JWT tokens.
type JWTSecret string

// RawIssuer is the raw CLI flag value for the issuer URL, injected from main.go via DI.
type RawIssuer string

// Issuer is the validated issuer URL for the OAuth authorization server.
type Issuer string

// AllowedUsers is the parsed and lowercased list of allowed GitHub usernames.
type AllowedUsers []string

// NewAllowedUsers parses a RawAllowedUsers value into an AllowedUsers list.
// It splits by comma, trims whitespace, drops empty entries, and lowercases each entry.
func NewAllowedUsers(raw RawAllowedUsers) AllowedUsers {
	if strings.TrimSpace(string(raw)) == "" {
		return AllowedUsers{}
	}

	parts := strings.Split(string(raw), ",")
	users := make(AllowedUsers, 0, len(parts))

	for _, part := range parts {
		entry := strings.TrimSpace(part)
		if entry == "" {
			continue
		}

		users = append(users, strings.ToLower(entry))
	}

	return users
}

// Contains reports whether the given username is in the allowed list.
// The comparison is case-insensitive.
func (u AllowedUsers) Contains(username string) bool {
	return slices.Contains(u, strings.ToLower(username))
}

// NewValidatedIssuer validates that the issuer string is a URL with an http or https
// scheme and a non-empty host.
func NewValidatedIssuer(raw RawIssuer) (Issuer, error) {
	parsed, err := url.Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("invalid issuer URL: %w", err)
	}

	if parsed.Scheme != schemeHTTP && parsed.Scheme != schemeHTTPS {
		return "", errIssuerMissingScheme
	}

	if parsed.Hostname() == "" {
		return "", errIssuerEmptyHost
	}

	cleanPath := strings.TrimRight(parsed.Path, "/")
	if cleanPath != "" {
		return "", errIssuerHasPath
	}

	if parsed.RawQuery != "" {
		return "", errIssuerHasQuery
	}

	if parsed.Fragment != "" {
		return "", errIssuerHasFragment
	}

	return Issuer(strings.TrimRight(string(raw), "/")), nil
}

// NewJWTSecret validates or auto-generates a JWT signing secret.
// If the input is empty, a cryptographically random 32-byte secret is generated
// and returned as a base64url-encoded string. If provided, the secret must be at
// least 32 characters for HMAC-SHA256 security.
func NewJWTSecret(raw RawJWTSecret) (JWTSecret, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed != "" {
		if len(trimmed) < jwtSecretLength {
			return "", fmt.Errorf(
				"%w: must be at least %d characters, got %d",
				errJWTSecretTooShort, jwtSecretLength, len(trimmed),
			)
		}

		return JWTSecret(trimmed), nil
	}

	buf := make([]byte, jwtSecretLength)

	_, err := rand.Read(buf)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT secret: %w", err)
	}

	return JWTSecret(base64.RawURLEncoding.EncodeToString(buf)), nil
}
