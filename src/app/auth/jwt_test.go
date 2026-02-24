package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

func TestIssueAccessToken_ValidToken(t *testing.T) {
	t.Parallel()

	secret := JWTSecret("test-secret-key-for-signing")
	issuer := Issuer("https://auth.example.com")

	tokenStr, err := IssueAccessToken(secret, issuer, 15*time.Minute, "alice", []string{"mcp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	parsed, err := jwt.Parse(tokenStr, func(_ *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse issued token: %v", err)
	}

	if !parsed.Valid {
		t.Fatal("expected valid token")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected MapClaims")
	}

	if claims["iss"] != "https://auth.example.com" {
		t.Fatalf("expected iss 'https://auth.example.com', got %v", claims["iss"])
	}

	if claims["sub"] != "alice" {
		t.Fatalf("expected sub 'alice', got %v", claims["sub"])
	}

	if claims["scope"] != "mcp" {
		t.Fatalf("expected scope 'mcp', got %v", claims["scope"])
	}

	if claims["jti"] == nil || claims["jti"] == "" {
		t.Fatal("expected non-empty jti claim")
	}

	if claims["iat"] == nil {
		t.Fatal("expected iat claim to be set")
	}

	if claims["exp"] == nil {
		t.Fatal("expected exp claim to be set")
	}
}

func TestIssueAccessToken_MultipleScopes(t *testing.T) {
	t.Parallel()

	secret := JWTSecret("test-secret")
	issuer := Issuer("https://auth.example.com")

	tokenStr, err := IssueAccessToken(secret, issuer, 15*time.Minute, "bob", []string{"mcp", "read", "write"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, err := jwt.Parse(tokenStr, func(_ *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	claims := parsed.Claims.(jwt.MapClaims)

	if claims["scope"] != "mcp read write" {
		t.Fatalf("expected scope 'mcp read write', got %v", claims["scope"])
	}
}

func TestIssueAccessToken_UniqueJTI(t *testing.T) {
	t.Parallel()

	secret := JWTSecret("test-secret")
	issuer := Issuer("https://auth.example.com")

	token1, err := IssueAccessToken(secret, issuer, 15*time.Minute, "alice", []string{"mcp"})
	if err != nil {
		t.Fatalf("unexpected error issuing token 1: %v", err)
	}

	token2, err := IssueAccessToken(secret, issuer, 15*time.Minute, "alice", []string{"mcp"})
	if err != nil {
		t.Fatalf("unexpected error issuing token 2: %v", err)
	}

	parsed1, _ := jwt.Parse(token1, func(_ *jwt.Token) (any, error) {
		return []byte(secret), nil
	})

	parsed2, _ := jwt.Parse(token2, func(_ *jwt.Token) (any, error) {
		return []byte(secret), nil
	})

	claims1 := parsed1.Claims.(jwt.MapClaims)
	claims2 := parsed2.Claims.(jwt.MapClaims)

	if claims1["jti"] == claims2["jti"] {
		t.Fatal("expected unique JTI values for different tokens")
	}
}

func TestIssueRefreshToken_NonEmpty(t *testing.T) {
	t.Parallel()

	token, err := IssueRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty refresh token")
	}
}

func TestIssueRefreshToken_UniqueTokens(t *testing.T) {
	t.Parallel()

	token1, err := IssueRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error issuing token 1: %v", err)
	}

	token2, err := IssueRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error issuing token 2: %v", err)
	}

	if token1 == token2 {
		t.Fatal("expected unique refresh tokens")
	}
}

func TestNewTokenVerifier_ValidToken(t *testing.T) {
	t.Parallel()

	secret := JWTSecret("verifier-test-secret")
	issuer := Issuer("https://auth.example.com")

	tokenStr, err := IssueAccessToken(secret, issuer, 15*time.Minute, "alice", []string{"mcp"})
	if err != nil {
		t.Fatalf("unexpected error issuing token: %v", err)
	}

	verifier := NewTokenVerifier(secret, issuer)

	info, err := verifier(context.Background(), tokenStr, nil)
	if err != nil {
		t.Fatalf("unexpected verification error: %v", err)
	}

	if info.UserID != "alice" {
		t.Fatalf("expected UserID 'alice', got %q", info.UserID)
	}

	if len(info.Scopes) != 1 || info.Scopes[0] != "mcp" {
		t.Fatalf("expected scopes [mcp], got %v", info.Scopes)
	}

	if info.Expiration.IsZero() {
		t.Fatal("expected non-zero expiration")
	}
}

func TestNewTokenVerifier_MultipleScopes(t *testing.T) {
	t.Parallel()

	secret := JWTSecret("verifier-test-secret")
	issuer := Issuer("https://auth.example.com")

	tokenStr, err := IssueAccessToken(secret, issuer, 15*time.Minute, "bob", []string{"mcp", "read", "write"})
	if err != nil {
		t.Fatalf("unexpected error issuing token: %v", err)
	}

	verifier := NewTokenVerifier(secret, issuer)

	info, err := verifier(context.Background(), tokenStr, nil)
	if err != nil {
		t.Fatalf("unexpected verification error: %v", err)
	}

	expectedScopes := []string{"mcp", "read", "write"}
	if len(info.Scopes) != len(expectedScopes) {
		t.Fatalf("expected %d scopes, got %d", len(expectedScopes), len(info.Scopes))
	}

	for i, s := range expectedScopes {
		if info.Scopes[i] != s {
			t.Fatalf("scope[%d]: expected %q, got %q", i, s, info.Scopes[i])
		}
	}
}

func TestNewTokenVerifier_ExpiredToken(t *testing.T) {
	t.Parallel()

	secret := JWTSecret("verifier-test-secret")
	issuer := Issuer("https://auth.example.com")

	now := time.Now().Add(-30 * time.Minute)

	claims := jwt.MapClaims{
		"iss":   string(issuer),
		"sub":   "alice",
		"exp":   jwt.NewNumericDate(now.Add(1 * time.Minute)),
		"iat":   jwt.NewNumericDate(now),
		"jti":   "test-jti",
		"scope": "mcp",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("unexpected error signing token: %v", err)
	}

	verifier := NewTokenVerifier(secret, issuer)

	_, err = verifier(context.Background(), tokenStr, nil)
	if err == nil {
		t.Fatal("expected error for expired token")
	}

	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected error to wrap auth.ErrInvalidToken, got %v", err)
	}
}

func TestNewTokenVerifier_WrongSecret(t *testing.T) {
	t.Parallel()

	issuingSecret := JWTSecret("correct-secret")
	verifyingSecret := JWTSecret("wrong-secret")
	issuer := Issuer("https://auth.example.com")

	tokenStr, err := IssueAccessToken(issuingSecret, issuer, 15*time.Minute, "alice", []string{"mcp"})
	if err != nil {
		t.Fatalf("unexpected error issuing token: %v", err)
	}

	verifier := NewTokenVerifier(verifyingSecret, issuer)

	_, err = verifier(context.Background(), tokenStr, nil)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}

	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected error to wrap auth.ErrInvalidToken, got %v", err)
	}
}

func TestNewTokenVerifier_InvalidTokenFormat(t *testing.T) {
	t.Parallel()

	secret := JWTSecret("test-secret")
	issuer := Issuer("https://auth.example.com")

	verifier := NewTokenVerifier(secret, issuer)

	_, err := verifier(context.Background(), "not-a-valid-jwt", nil)
	if err == nil {
		t.Fatal("expected error for invalid token format")
	}

	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected error to wrap auth.ErrInvalidToken, got %v", err)
	}
}

func TestNewTokenVerifier_WrongIssuer(t *testing.T) {
	t.Parallel()

	secret := JWTSecret("test-secret")
	issuingIssuer := Issuer("https://auth.example.com")
	verifyingIssuer := Issuer("https://other.example.com")

	tokenStr, err := IssueAccessToken(secret, issuingIssuer, 15*time.Minute, "alice", []string{"mcp"})
	if err != nil {
		t.Fatalf("unexpected error issuing token: %v", err)
	}

	verifier := NewTokenVerifier(secret, verifyingIssuer)

	_, err = verifier(context.Background(), tokenStr, nil)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}

	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected error to wrap auth.ErrInvalidToken, got %v", err)
	}
}

func TestNewTokenVerifier_EmptyScopes(t *testing.T) {
	t.Parallel()

	secret := JWTSecret("test-secret")
	issuer := Issuer("https://auth.example.com")

	tokenStr, err := IssueAccessToken(secret, issuer, 15*time.Minute, "alice", nil)
	if err != nil {
		t.Fatalf("unexpected error issuing token: %v", err)
	}

	verifier := NewTokenVerifier(secret, issuer)

	info, err := verifier(context.Background(), tokenStr, nil)
	if err != nil {
		t.Fatalf("unexpected verification error: %v", err)
	}

	if len(info.Scopes) != 0 {
		t.Fatalf("expected empty scopes, got %v", info.Scopes)
	}
}

func TestNewTokenVerifier_RejectsNonHS256(t *testing.T) {
	t.Parallel()

	secret := JWTSecret("test-secret")
	issuer := Issuer("https://auth.example.com")

	claims := jwt.MapClaims{
		"iss":   string(issuer),
		"sub":   "alice",
		"exp":   jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		"iat":   jwt.NewNumericDate(time.Now()),
		"jti":   "test-jti",
		"scope": "mcp",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)

	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("unexpected error signing token: %v", err)
	}

	verifier := NewTokenVerifier(secret, issuer)

	_, err = verifier(context.Background(), tokenStr, nil)
	if err == nil {
		t.Fatal("expected error for non-HS256 signing method")
	}

	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected error to wrap auth.ErrInvalidToken, got %v", err)
	}
}
