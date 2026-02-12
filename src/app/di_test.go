package app

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/0xalexb/intervals-icu-mcp/src/app/client"
)

func TestModule_ProvidesServer(t *testing.T) {
	t.Parallel()

	var server *Server

	serverTransport, _ := mcp.NewInMemoryTransports()

	testClient := client.NewTestClient("http://localhost", "test-key", "i123", nil)

	app := fxtest.New(t,
		fx.Decorate(func() mcp.Transport { return serverTransport }),
		fx.Decorate(func() *client.Client { return testClient }),
		Module,
		fx.Populate(&server),
	)

	if server == nil {
		t.Fatal("expected Module to provide a non-nil Server")
	}

	if server.mcpServer == nil {
		t.Fatal("expected Server to have a non-nil mcp server")
	}

	app.RequireStart()
	app.RequireStop()
}
