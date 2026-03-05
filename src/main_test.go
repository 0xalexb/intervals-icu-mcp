package main

import (
	"context"
	"os"
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

func TestResolveSecretsFromEnv(t *testing.T) {
	t.Run("env vars fill empty flags", func(t *testing.T) {
		t.Setenv("GITHUB_CLIENT_SECRET", "gh-secret-from-env")
		t.Setenv("JWT_SECRET", "jwt-secret-from-env")

		flags := cliFlags{}
		resolveSecretsFromEnv(&flags)

		if flags.githubClientSec != "gh-secret-from-env" {
			t.Errorf("expected githubClientSec %q, got %q", "gh-secret-from-env", flags.githubClientSec)
		}

		if flags.jwtSecret != "jwt-secret-from-env" {
			t.Errorf("expected jwtSecret %q, got %q", "jwt-secret-from-env", flags.jwtSecret)
		}
	})

	t.Run("cli flags take precedence over env vars", func(t *testing.T) {
		t.Setenv("GITHUB_CLIENT_SECRET", "gh-secret-from-env")
		t.Setenv("JWT_SECRET", "jwt-secret-from-env")

		flags := cliFlags{
			githubClientSec: "gh-secret-from-flag",
			jwtSecret:       "jwt-secret-from-flag",
		}
		resolveSecretsFromEnv(&flags)

		if flags.githubClientSec != "gh-secret-from-flag" {
			t.Errorf("expected githubClientSec %q, got %q", "gh-secret-from-flag", flags.githubClientSec)
		}

		if flags.jwtSecret != "jwt-secret-from-flag" {
			t.Errorf("expected jwtSecret %q, got %q", "jwt-secret-from-flag", flags.jwtSecret)
		}
	})

	t.Run("unset env vars leave flags empty", func(t *testing.T) {
		os.Unsetenv("GITHUB_CLIENT_SECRET")
		os.Unsetenv("JWT_SECRET")

		flags := cliFlags{}
		resolveSecretsFromEnv(&flags)

		if flags.githubClientSec != "" {
			t.Errorf("expected empty githubClientSec, got %q", flags.githubClientSec)
		}

		if flags.jwtSecret != "" {
			t.Errorf("expected empty jwtSecret, got %q", flags.jwtSecret)
		}
	})
}
