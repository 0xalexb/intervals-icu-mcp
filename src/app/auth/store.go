package auth

import (
	"errors"
	"sync"
	"time"
)

var (
	errAuthCodeNotFound      = errors.New("authorization code not found")
	errAuthCodeExpired       = errors.New("authorization code has expired")
	errRefreshTokenNotFound  = errors.New("refresh token not found")
	errRefreshTokenExpired   = errors.New("refresh token has expired")
	errClientNotFound        = errors.New("client not found")
)

// AuthCode represents an OAuth 2.1 authorization code stored in memory.
type AuthCode struct {
	Code                string
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	GitHubUsername       string
	Scopes              []string
	ExpiresAt           time.Time
}

// RefreshToken represents an OAuth 2.1 refresh token stored in memory.
type RefreshToken struct {
	Token          string
	ClientID       string
	GitHubUsername  string
	Scopes         []string
	ExpiresAt      time.Time
}

// RegisteredClient represents a dynamically registered OAuth client.
type RegisteredClient struct {
	ClientID     string
	RedirectURIs []string
	ClientName   string
	GrantTypes   []string
	CreatedAt    time.Time
}

// Store is a thread-safe in-memory store for OAuth entities.
type Store struct {
	mu            sync.RWMutex
	authCodes     map[string]*AuthCode
	refreshTokens map[string]*RefreshToken
	clients       map[string]*RegisteredClient
}

// NewStore creates a new empty Store.
func NewStore() *Store {
	return &Store{
		authCodes:     make(map[string]*AuthCode),
		refreshTokens: make(map[string]*RefreshToken),
		clients:       make(map[string]*RegisteredClient),
	}
}

// SaveAuthCode stores an authorization code.
func (s *Store) SaveAuthCode(code *AuthCode) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.authCodes[code.Code] = code
}

// ConsumeAuthCode retrieves and deletes an authorization code (one-time use).
// Returns an error if the code is not found or has expired.
func (s *Store) ConsumeAuthCode(code string, now time.Time) (*AuthCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ac, ok := s.authCodes[code]
	if !ok {
		return nil, errAuthCodeNotFound
	}

	delete(s.authCodes, code)

	if now.After(ac.ExpiresAt) {
		return nil, errAuthCodeExpired
	}

	return ac, nil
}

// SaveRefreshToken stores a refresh token.
func (s *Store) SaveRefreshToken(token *RefreshToken) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.refreshTokens[token.Token] = token
}

// ConsumeRefreshToken retrieves and deletes a refresh token (rotation).
// The caller is responsible for saving a new refresh token after consuming the old one.
// Returns an error if the token is not found or has expired.
func (s *Store) ConsumeRefreshToken(token string, now time.Time) (*RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rt, ok := s.refreshTokens[token]
	if !ok {
		return nil, errRefreshTokenNotFound
	}

	delete(s.refreshTokens, token)

	if now.After(rt.ExpiresAt) {
		return nil, errRefreshTokenExpired
	}

	return rt, nil
}

// SaveClient stores a registered client.
func (s *Store) SaveClient(client *RegisteredClient) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[client.ClientID] = client
}

// GetClient retrieves a registered client by its client ID.
// Returns an error if the client is not found.
func (s *Store) GetClient(clientID string) (*RegisteredClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, ok := s.clients[clientID]
	if !ok {
		return nil, errClientNotFound
	}

	return client, nil
}
