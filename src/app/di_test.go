package app

import (
	"net/http"
	"testing"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/0xalexb/intervals-icu-mcp/src/app/client"
)

func TestModule_ProvidesServer(t *testing.T) {
	t.Parallel()

	var server *Server
	var handler http.Handler

	testClient := client.NewTestClient("http://localhost", "test-key", "i123", nil)

	app := fxtest.New(t,
		fx.Supply(TransportStreamable),
		fx.Decorate(func() *client.Client { return testClient }),
		Module,
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
