package github

import "go.uber.org/fx"

// Module provides the GitHub OAuth client via DI.
var Module = fx.Module("github", //nolint:gochecknoglobals // fx.Module as package variable is the standard DI pattern.
	fx.Provide(NewClient),
)
