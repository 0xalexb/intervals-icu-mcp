package tools

import (
	"testing"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/0xalexb/intervals-icu-mcp/src/app/client"
)

func TestModule_ProvidesToolRegistrations(t *testing.T) {
	t.Parallel()

	var registrations []ToolRegistration

	testClient := client.NewTestClient("http://localhost", "test-key", "i123", nil)

	app := fxtest.New(t,
		Module,
		fx.Supply(testClient),
		fx.Invoke(fx.Annotate(
			func(regs []ToolRegistration) {
				registrations = regs
			},
			fx.ParamTags(`group:"mcp_tools"`),
		)),
	)

	app.RequireStart()

	if len(registrations) == 0 {
		t.Fatal("expected Module to provide at least one ToolRegistration")
	}

	app.RequireStop()
}
