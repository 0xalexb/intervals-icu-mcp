package app

import (
	"net/http"
	"testing"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/0xalexb/intervals-icu-mcp/src/app/api"
	"github.com/0xalexb/intervals-icu-mcp/src/app/auth"
	"github.com/0xalexb/intervals-icu-mcp/src/app/clients/intervals"
)

func TestModule_ProvidesServer_Streamable(t *testing.T) {
	t.Parallel()

	var server *Server
	var handler http.Handler

	testClient := intervals.NewTestClient("http://localhost", "test-key", "i123", nil)

	app := fxtest.New(t,
		fx.Supply(TransportStreamable),
		fx.Supply(api.RawAllowedOrigins("http://localhost:3000,http://127.0.0.1:8080,http://[::1]:9090")),
		fx.Supply(auth.GitHubClientID("test-client-id")),
		fx.Supply(auth.GitHubClientSecret("test-client-secret")),
		fx.Supply(auth.RawAllowedUsers("")),
		fx.Supply(auth.RawJWTSecret("")),
		fx.Supply(auth.RawIssuer("http://localhost:8080")),
		fx.Decorate(func() *intervals.Client { return testClient }),
		Module,
		StreamableModules,
		fx.Populate(&server),
		fx.Populate(fx.Annotate(&handler, fx.ParamTags(`name:"mcp"`))),
	)

	if server == nil {
		t.Fatal("expected Module to provide a non-nil Server")
	}

	if server.mcpServer == nil {
		t.Fatal("expected Server to have a non-nil mcp server")
	}

	if handler == nil {
		t.Fatal("expected Module to provide a non-nil HTTP handler via api module")
	}

	app.RequireStart()
	app.RequireStop()
}

func TestModule_ProvidesServer_Stdio(t *testing.T) {
	t.Parallel()

	var server *Server

	testClient := intervals.NewTestClient("http://localhost", "test-key", "i123", nil)

	app := fxtest.New(t,
		fx.Supply(TransportStdio),
		fx.Decorate(func() *intervals.Client { return testClient }),
		Module,
		fx.Populate(&server),
	)

	if server == nil {
		t.Fatal("expected Module to provide a non-nil Server")
	}

	app.RequireStart()
	app.RequireStop()
}
