package auth

import (
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// AuthorizationServerMetadata represents the OAuth 2.0 Authorization Server Metadata
// per RFC 8414. Uses a project-local type rather than the SDK's oauthex.AuthServerMeta
// to include only the fields this server supports.
type AuthorizationServerMetadata struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	RegistrationEndpoint             string   `json:"registration_endpoint,omitempty"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	GrantTypesSupported              []string `json:"grant_types_supported,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	CodeChallengeMethodsSupported    []string `json:"code_challenge_methods_supported,omitempty"`
	ScopesSupported                  []string `json:"scopes_supported,omitempty"`
}

// NewAuthorizationServerMetadata constructs authorization server metadata with
// endpoints derived from the issuer URL.
func NewAuthorizationServerMetadata(issuer Issuer) *AuthorizationServerMetadata {
	base := string(issuer)

	return &AuthorizationServerMetadata{
		Issuer:                            base,
		AuthorizationEndpoint:             base + "/oauth/authorize",
		TokenEndpoint:                     base + "/oauth/token",
		RegistrationEndpoint:              base + "/oauth/register",
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported: []string{"none"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		ScopesSupported:                   []string{"mcp"},
	}
}

// NewProtectedResourceMetadata constructs protected resource metadata per RFC 9728
// for the MCP server, indicating the issuer as the authorization server and supporting
// bearer token authentication via the Authorization header.
func NewProtectedResourceMetadata(issuer Issuer) *oauthex.ProtectedResourceMetadata {
	base := string(issuer)

	return &oauthex.ProtectedResourceMetadata{
		Resource:               base,
		AuthorizationServers:   []string{base},
		ScopesSupported:        []string{"mcp"},
		BearerMethodsSupported: []string{"header"},
	}
}
