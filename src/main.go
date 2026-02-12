// Package main is the entry point for the intervals-icu-mcp server.
package main

import (
	"flag"
	"os"

	di "github.com/0xalexb/hjarta-di"

	"github.com/0xalexb/intervals-icu-mcp/src/app"
)

func main() {
	showVersion := flag.Bool("version", false, "Print the application version and exit.")
	flag.BoolVar(showVersion, "v", false, "Print the application version and exit (shorthand).")
	flag.Parse()

	if *showVersion {
		_, _ = os.Stdout.WriteString(di.Version + "\n")

		return
	}

	application := di.NewApp(
		di.WithLogLevel("info"),
		di.WithModules(app.Module),
	)

	application.Run()
}
