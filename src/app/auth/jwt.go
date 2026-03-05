package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

const refreshTokenLength = 32

var errTokenVerification = errors.New("token verification failed")

// IssueAccessToken creates a signed JWT access token using HMAC-SHA256.
// The token includes standard claims: iss, sub (username), exp, iat, jti (uuid),
// and scope (space-separated scopes).
func IssueAccessToken(
	secret JWTSecret,
	issuer Issuer,
	ttl time.Duration,
	username string,
	scopes []string,
) (string, error) {
	jti, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("generating JTI: %w", err)
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"iss":   string(issuer),
		"sub":   username,
		"exp":   jwt.NewNumericDate(now.Add(ttl)),
		"nbf":   jwt.NewNumericDate(now),
		"iat":   jwt.NewNumericDate(now),
		"jti":   jti.String(),
		"scope": strings.Join(scopes, " "),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signed, nil
}

// IssueRefreshToken generates a cryptographically random 32-byte base64url-encoded
// refresh token.
func IssueRefreshToken() (string, error) {
	buf := make([]byte, refreshTokenLength)

	_, err := rand.Read(buf)
	if err != nil {
		return "", fmt.Errorf("generating refresh token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// NewTokenVerifier creates an auth.TokenVerifier that validates JWT access tokens
// signed with the given secret and issuer. It parses the JWT, validates the signature
// and standard claims, and maps the result to auth.TokenInfo with UserID set to
// the GitHub username from the sub claim.
func NewTokenVerifier(secret JWTSecret, issuer Issuer) auth.TokenVerifier {
	return func(_ context.Context, tokenString string, _ *http.Request) (*auth.TokenInfo, error) {
		token, err := jwt.Parse(tokenString, func(_ *jwt.Token) (any, error) {
			return []byte(secret), nil
		},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithIssuer(string(issuer)),
			jwt.WithExpirationRequired(),
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", auth.ErrInvalidToken, err)
		}

		if !token.Valid {
			return nil, fmt.Errorf("%w: %w", auth.ErrInvalidToken, errTokenVerification)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("%w: unexpected claims type", auth.ErrInvalidToken)
		}

		sub, hasSub := claims["sub"].(string)
		if !hasSub || sub == "" {
			return nil, fmt.Errorf("%w: missing sub claim", auth.ErrInvalidToken)
		}

		info := &auth.TokenInfo{
			UserID: sub,
		}

		if exp, hasExp := claims["exp"].(float64); hasExp {
			info.Expiration = time.Unix(int64(exp), 0)
		}

		if scope, hasScope := claims["scope"].(string); hasScope && scope != "" {
			info.Scopes = strings.Split(scope, " ")
		}

		return info, nil
	}
}
