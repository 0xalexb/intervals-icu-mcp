package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExchangeCode_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Error("expected Accept: application/json header")
		}

		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Error("expected Content-Type: application/x-www-form-urlencoded header")
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}

		if r.FormValue("client_id") != "test-client-id" {
			t.Errorf("expected client_id 'test-client-id', got %q", r.FormValue("client_id"))
		}

		if r.FormValue("client_secret") != "test-client-secret" {
			t.Errorf("expected client_secret 'test-client-secret', got %q", r.FormValue("client_secret"))
		}

		if r.FormValue("code") != "test-code" {
			t.Errorf("expected code 'test-code', got %q", r.FormValue("code"))
		}

		w.Header().Set("Content-Type", "application/json")

		resp := map[string]string{"access_token": "gho_test_token_123"}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	gh := NewTestClient(srv.URL, "")

	token, err := gh.ExchangeCode(
		context.Background(),
		"test-client-id",
		"test-client-secret",
		"test-code",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token != "gho_test_token_123" {
		t.Fatalf("expected token 'gho_test_token_123', got %q", token)
	}
}

func TestExchangeCode_GitHubError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		resp := map[string]string{
			"error":             "bad_verification_code",
			"error_description": "The code passed is incorrect or expired.",
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	gh := NewTestClient(srv.URL, "")

	_, err := gh.ExchangeCode(
		context.Background(),
		"test-client-id",
		"test-client-secret",
		"bad-code",
	)
	if err == nil {
		t.Fatal("expected error for bad verification code")
	}

	if !errors.Is(err, ErrGitHubTokenExchange) {
		t.Fatalf("expected ErrGitHubTokenExchange, got %v", err)
	}
}

func TestExchangeCode_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	gh := NewTestClient(srv.URL, "")

	_, err := gh.ExchangeCode(
		context.Background(),
		"test-client-id",
		"test-client-secret",
		"test-code",
	)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}

	if !errors.Is(err, ErrGitHubTokenExchange) {
		t.Fatalf("expected ErrGitHubTokenExchange, got %v", err)
	}
}

func TestExchangeCode_EmptyAccessToken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		resp := map[string]string{"access_token": ""}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	gh := NewTestClient(srv.URL, "")

	_, err := gh.ExchangeCode(
		context.Background(),
		"test-client-id",
		"test-client-secret",
		"test-code",
	)
	if err == nil {
		t.Fatal("expected error for empty access token")
	}

	if !errors.Is(err, ErrGitHubTokenExchange) {
		t.Fatalf("expected ErrGitHubTokenExchange, got %v", err)
	}
}

func TestExchangeCode_InvalidJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	gh := NewTestClient(srv.URL, "")

	_, err := gh.ExchangeCode(
		context.Background(),
		"test-client-id",
		"test-client-secret",
		"test-code",
	)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	if !errors.Is(err, ErrGitHubTokenExchange) {
		t.Fatalf("expected ErrGitHubTokenExchange, got %v", err)
	}
}

func TestGetUser_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer gho_test_token" {
			t.Errorf("expected Authorization 'Bearer gho_test_token', got %q", r.Header.Get("Authorization"))
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Error("expected Accept: application/json header")
		}

		w.Header().Set("Content-Type", "application/json")

		resp := map[string]any{
			"login": "alice",
			"id":    12345,
			"name":  "Alice",
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	gh := NewTestClient("", srv.URL)

	user, err := gh.GetUser(context.Background(), "gho_test_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.Login != "alice" {
		t.Fatalf("expected login 'alice', got %q", user.Login)
	}
}

func TestGetUser_Unauthorized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()

	gh := NewTestClient("", srv.URL)

	_, err := gh.GetUser(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}

	if !errors.Is(err, ErrGitHubUserFetch) {
		t.Fatalf("expected ErrGitHubUserFetch, got %v", err)
	}
}

func TestGetUser_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	gh := NewTestClient("", srv.URL)

	_, err := gh.GetUser(context.Background(), "gho_test_token")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}

	if !errors.Is(err, ErrGitHubUserFetch) {
		t.Fatalf("expected ErrGitHubUserFetch, got %v", err)
	}
}

func TestGetUser_EmptyLogin(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		resp := map[string]any{
			"login": "",
			"id":    12345,
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	gh := NewTestClient("", srv.URL)

	_, err := gh.GetUser(context.Background(), "gho_test_token")
	if err == nil {
		t.Fatal("expected error for empty login")
	}

	if !errors.Is(err, ErrGitHubUserFetch) {
		t.Fatalf("expected ErrGitHubUserFetch, got %v", err)
	}
}

func TestGetUser_InvalidJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	gh := NewTestClient("", srv.URL)

	_, err := gh.GetUser(context.Background(), "gho_test_token")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	if !errors.Is(err, ErrGitHubUserFetch) {
		t.Fatalf("expected ErrGitHubUserFetch, got %v", err)
	}
}

func TestExchangeCode_ConnectionRefused(t *testing.T) {
	t.Parallel()

	gh := NewTestClient("http://127.0.0.1:1", "")

	_, err := gh.ExchangeCode(
		context.Background(),
		"test-client-id",
		"test-client-secret",
		"test-code",
	)
	if err == nil {
		t.Fatal("expected error for connection refused")
	}

	if !errors.Is(err, ErrGitHubTokenExchange) {
		t.Fatalf("expected ErrGitHubTokenExchange, got %v", err)
	}
}

func TestGetUser_ConnectionRefused(t *testing.T) {
	t.Parallel()

	gh := NewTestClient("", "http://127.0.0.1:1")

	_, err := gh.GetUser(context.Background(), "gho_test_token")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}

	if !errors.Is(err, ErrGitHubUserFetch) {
		t.Fatalf("expected ErrGitHubUserFetch, got %v", err)
	}
}

func TestExchangeCode_CancelledContext(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"access_token": "token"})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	gh := NewTestClient(srv.URL, "")

	_, err := gh.ExchangeCode(ctx, "id", "secret", "code")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestGetUser_CancelledContext(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"login": "alice"})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	gh := NewTestClient("", srv.URL)

	_, err := gh.GetUser(ctx, "token")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
