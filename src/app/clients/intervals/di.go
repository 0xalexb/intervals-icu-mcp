package intervals

import (
	"os"

	"go.uber.org/fx"
)

// Module provides the Intervals.icu API client and its configuration.
var Module = fx.Module("intervals", //nolint:gochecknoglobals // standard DI pattern.
	fx.Provide(func() Config {
		return Config{
			APIKey:    os.Getenv("INTERVALS_API_KEY"),
			AthleteID: os.Getenv("INTERVALS_ATHLETE_ID"),
		}
	}),
	fx.Provide(NewClient),
)
