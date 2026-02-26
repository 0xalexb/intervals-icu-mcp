package rest

import "go.uber.org/fx"

// Module provides the OAuth HTTP handler via DI.
var Module = fx.Module("rest", //nolint:gochecknoglobals // fx.Module as package variable is the standard DI pattern.
	fx.Provide(NewHandler),
)
