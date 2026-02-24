package auth

import (
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"go.uber.org/fx"
)

// Module provides all OAuth authentication components via DI.
var Module = fx.Module("auth", //nolint:gochecknoglobals // fx.Module as package variable is the standard DI pattern.
	fx.Provide(NewAllowedUsers),
	fx.Provide(NewValidatedIssuer),
	fx.Provide(NewJWTSecret),
	fx.Provide(NewStore),
	fx.Provide(NewHandler),
	fx.Provide(NewGitHubClient),
	fx.Provide(NewAuthorizationServerMetadata),
	fx.Provide(NewProtectedResourceMetadata),
	fx.Provide(NewTokenVerifier),
)

// ProtectedResourceMetadata re-exports the type for use in other packages.
type ProtectedResourceMetadata = oauthex.ProtectedResourceMetadata

// TokenVerifier re-exports the type for use in other packages.
type TokenVerifier = auth.TokenVerifier
