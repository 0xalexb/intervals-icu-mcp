package client

import (
	"os"

	"go.uber.org/fx"
)

// Module provides the Intervals.icu API client and its configuration.
var Module = fx.Module("client", //nolint:gochecknoglobals // fx.Module as package variable is the standard DI pattern.
	fx.Provide(func() Config {
		return Config{
			APIKey:    os.Getenv("INTERVALS_API_KEY"),
			AthleteID: os.Getenv("INTERVALS_ATHLETE_ID"),
		}
	}),
	fx.Provide(NewClient),
)
