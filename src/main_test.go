package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	t.Parallel()

	binary := t.TempDir() + "/intervals-icu-mcp"
	ldflags := "-X github.com/0xalexb/hjarta-di.Version=1.2.3-test"
	buildCmd := exec.CommandContext(context.Background(), "go", "build", "-ldflags", ldflags, "-o", binary, "./src/")
	buildCmd.Dir = ".."

	out, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, out)
	}

	tests := []struct {
		name string
		flag string
	}{
		{name: "long flag", flag: "-version"},
		{name: "short flag", flag: "-v"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := exec.CommandContext(context.Background(), binary, tt.flag)

			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("binary exited with error: %v", err)
			}

			got := strings.TrimSpace(string(output))
			if got != "1.2.3-test" {
				t.Errorf("expected version output %q, got %q", "1.2.3-test", got)
			}
		})
	}
}
