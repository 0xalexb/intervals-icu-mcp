// Package main is the entry point for the intervals-icu-mcp server.
package main

import (
	"flag"
	"fmt"
	"os"

	di "github.com/0xalexb/hjarta-di"
	"github.com/0xalexb/hjarta-di/listener"
	"go.uber.org/fx"

	"github.com/0xalexb/intervals-icu-mcp/src/app"
	"github.com/0xalexb/intervals-icu-mcp/src/app/api"
)

const (
	defaultAddress        = "127.0.0.1:8080"
	defaultAllowedOrigins = ""
	listenerName          = "mcp"
)

func main() {
	var (
		showVersion    bool
		address        string
		transport      string
		allowedOrigins string
	)

	flag.BoolVar(&showVersion, "version", false, "Print the application version and exit.")
	flag.BoolVar(&showVersion, "v", false, "Print the application version and exit (shorthand).")
	flag.StringVar(&transport, "transport", string(app.TransportStdio), "Transport type: stdio or streamable.")
	flag.StringVar(&address, "address", defaultAddress,
		"Listen address for streamable HTTP transport (e.g., :8080 or 127.0.0.1:9000).")
	flag.StringVar(&allowedOrigins, "allowed-origins", defaultAllowedOrigins,
		"Comma-separated list of allowed CORS origins as full URLs (e.g., http://localhost:3000,https://example.com).")
	flag.Parse()

	if showVersion {
		_, _ = os.Stdout.WriteString(di.Version + "\n")

		return
	}

	transportValue := app.Transport(transport)

	if transportValue != app.TransportStdio && transportValue != app.TransportStreamable {
		_, _ = fmt.Fprintf(os.Stderr, "invalid transport %q: must be %q or %q\n",
			transport, app.TransportStdio, app.TransportStreamable)

		os.Exit(1)
	}

	opts := []di.Option{
		di.WithLogLevel("info"),
		di.WithModules(
			app.Module,
			fx.Supply(transportValue),
			fx.Supply(api.RawAllowedOrigins(allowedOrigins)),
		),
	}

	if transportValue == app.TransportStreamable {
		opts = append(opts, di.WithHTTPListener(listenerName, listener.WithAddress(address)))
	}

	application := di.NewApp(opts...)

	application.Run()
}
