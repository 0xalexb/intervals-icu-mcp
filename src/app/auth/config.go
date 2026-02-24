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
	errIssuerMissingScheme = errors.New("issuer must have http or https scheme")
	errIssuerEmptyHost     = errors.New("issuer must have a non-empty host")
)

// GitHubClientID is the GitHub OAuth app client ID, injected from main.go via DI.
type GitHubClientID string

// GitHubClientSecret is the GitHub OAuth app client secret, injected from main.go via DI.
type GitHubClientSecret string

// RawAllowedUsers is the raw comma-separated flag value for allowed GitHub usernames,
// injected from main.go via DI.
type RawAllowedUsers string

// JWTSecret is the HMAC-SHA256 signing key for JWT tokens.
type JWTSecret string

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
func NewValidatedIssuer(raw Issuer) (Issuer, error) {
	parsed, err := url.Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("invalid issuer URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errIssuerMissingScheme
	}

	if parsed.Hostname() == "" {
		return "", errIssuerEmptyHost
	}

	return raw, nil
}

// NewJWTSecret validates or auto-generates a JWT signing secret.
// If the input is empty, a cryptographically random 32-byte secret is generated
// and returned as a base64url-encoded string.
func NewJWTSecret(raw JWTSecret) (JWTSecret, error) {
	if strings.TrimSpace(string(raw)) != "" {
		return raw, nil
	}

	buf := make([]byte, jwtSecretLength)

	_, err := rand.Read(buf)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT secret: %w", err)
	}

	return JWTSecret(base64.RawURLEncoding.EncodeToString(buf)), nil
}
