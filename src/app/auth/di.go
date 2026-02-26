package auth

import (
	"context"

	"go.uber.org/fx"
)

// Module provides all OAuth authentication components via DI.
var Module = fx.Module("auth", //nolint:gochecknoglobals // fx.Module as package variable is the standard DI pattern.
	fx.Provide(NewAllowedUsers),
	fx.Provide(NewValidatedIssuer),
	fx.Provide(NewJWTSecret),
	fx.Provide(NewStore),
	fx.Provide(NewAuthorizationServerMetadata),
	fx.Provide(NewProtectedResourceMetadata),
	fx.Provide(NewTokenVerifier),
	fx.Invoke(registerStoreCleanup),
)

func registerStoreCleanup(lc fx.Lifecycle, store *Store) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			store.StartCleanup()

			return nil
		},
		OnStop: func(_ context.Context) error {
			store.StopCleanup()

			return nil
		},
	})
}
