// Package main is the entry point for the intervals-icu-mcp server.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	di "github.com/0xalexb/hjarta-di"
	"github.com/0xalexb/hjarta-di/listener"
	"go.uber.org/fx"

	"github.com/0xalexb/intervals-icu-mcp/src/app"
	"github.com/0xalexb/intervals-icu-mcp/src/app/api"
	"github.com/0xalexb/intervals-icu-mcp/src/app/auth"
)

const (
	defaultAddress        = "127.0.0.1:8080"
	defaultAllowedOrigins = ""
	listenerName          = "mcp"
)

type cliFlags struct {
	showVersion    bool
	address        string
	transport      string
	allowedOrigins string
	githubClientID string
	githubClientSec string
	allowedUsers   string
	jwtSecret      string
	authIssuer     string
}

func parseFlags() cliFlags {
	var flags cliFlags

	flag.BoolVar(&flags.showVersion, "version", false,
		"Print the application version and exit.")
	flag.BoolVar(&flags.showVersion, "v", false,
		"Print the application version and exit (shorthand).")
	flag.StringVar(&flags.transport, "transport",
		string(app.TransportStdio), "Transport type: stdio or streamable.")
	flag.StringVar(&flags.address, "address", defaultAddress,
		"Listen address for streamable HTTP transport.")
	flag.StringVar(&flags.allowedOrigins, "allowed-origins", defaultAllowedOrigins,
		"Comma-separated allowed CORS origins as full URLs.")
	flag.StringVar(&flags.githubClientID, "github-client-id", "",
		"GitHub OAuth app client ID (required for streamable).")
	flag.StringVar(&flags.githubClientSec, "github-client-secret", "",
		"GitHub OAuth app client secret.")
	flag.StringVar(&flags.allowedUsers, "allowed-users", "",
		"Comma-separated list of allowed GitHub usernames.")
	flag.StringVar(&flags.jwtSecret, "jwt-secret", "",
		"HMAC-SHA256 signing key for JWT tokens (auto-generated if empty).")
	flag.StringVar(&flags.authIssuer, "auth-issuer", "",
		"Issuer URL for the OAuth authorization server (required for streamable).")
	flag.Parse()

	return flags
}

func main() {
	flags := parseFlags()

	if flags.showVersion {
		_, _ = os.Stdout.WriteString(di.Version + "\n")

		return
	}

	transportValue := app.Transport(flags.transport)

	if transportValue != app.TransportStdio && transportValue != app.TransportStreamable {
		_, _ = fmt.Fprintf(os.Stderr, "invalid transport %q: must be %q or %q\n",
			flags.transport, app.TransportStdio, app.TransportStreamable)

		os.Exit(1)
	}

	validateStreamableFlags(transportValue, flags)

	opts := []di.Option{
		di.WithLogLevel("info"),
		di.WithModules(
			app.Module,
			fx.Supply(transportValue),
		),
	}

	if transportValue == app.TransportStreamable {
		opts = append(opts,
			di.WithHTTPListener(listenerName, listener.WithAddress(flags.address)))
		opts = append(opts, di.WithModules(
			app.StreamableModules,
			fx.Supply(api.RawAllowedOrigins(flags.allowedOrigins)),
			fx.Supply(auth.GitHubClientID(flags.githubClientID)),
			fx.Supply(auth.GitHubClientSecret(flags.githubClientSec)),
			fx.Supply(auth.RawAllowedUsers(flags.allowedUsers)),
			fx.Supply(auth.RawJWTSecret(flags.jwtSecret)),
			fx.Supply(auth.RawIssuer(flags.authIssuer)),
		))
	}

	application := di.NewApp(opts...)

	application.Run()
}

func validateStreamableFlags(transport app.Transport, flags cliFlags) {
	if transport != app.TransportStreamable {
		return
	}

	type requiredFlag struct {
		name  string
		value string
	}

	required := []requiredFlag{
		{"--github-client-id", flags.githubClientID},
		{"--github-client-secret", flags.githubClientSec},
		{"--auth-issuer", flags.authIssuer},
	}

	var missing []string

	for _, f := range required {
		if f.value == "" {
			missing = append(missing, f.name)
		}
	}

	if len(missing) > 0 {
		_, _ = fmt.Fprintf(os.Stderr,
			"streamable transport requires flags: %s\n", strings.Join(missing, ", "))

		os.Exit(1)
	}
}
