package auth

import (
	"errors"
	"sync"
	"time"
)

const (
	cleanupInterval  = 5 * time.Minute
	maxClients       = 1000
	maxAuthCodes     = 10000
	maxRefreshTokens = 10000
	clientTTL        = 30 * 24 * time.Hour
)

var (
	errAuthCodeNotFound      = errors.New("authorization code not found")
	errAuthCodeExpired       = errors.New("authorization code has expired")
	errRefreshTokenNotFound = errors.New("refresh token not found")
	errClientNotFound        = errors.New("client not found")
	errMaxClientsReached     = errors.New("maximum number of registered clients reached")
	errMaxAuthCodesReached   = errors.New("maximum number of authorization codes reached")
	errMaxRefreshTokensReached = errors.New("maximum number of refresh tokens reached")
)

// Code represents an OAuth 2.1 authorization code stored in memory.
type Code struct {
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
// It runs a periodic cleanup goroutine to evict expired auth codes, refresh tokens, and clients.
type Store struct {
	mu            sync.RWMutex
	authCodes     map[string]*Code
	refreshTokens map[string]*RefreshToken
	clients       map[string]*RegisteredClient
	stopCleanup   chan struct{}
	stopOnce      sync.Once
}

// NewStore creates a new empty Store.
func NewStore() *Store {
	return &Store{
		authCodes:     make(map[string]*Code),
		refreshTokens: make(map[string]*RefreshToken),
		clients:       make(map[string]*RegisteredClient),
		stopCleanup:   make(chan struct{}),
	}
}

// StartCleanup begins periodic eviction of expired auth codes and refresh tokens.
func (s *Store) StartCleanup() {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.evictExpired(time.Now())
			case <-s.stopCleanup:
				return
			}
		}
	}()
}

// StopCleanup stops the periodic cleanup goroutine. It is safe to call multiple times.
func (s *Store) StopCleanup() {
	s.stopOnce.Do(func() {
		close(s.stopCleanup)
	})
}

// evictExpired removes all expired auth codes, refresh tokens, and clients.
func (s *Store) evictExpired(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.evictExpiredLocked(now)
}

// evictExpiredLocked is the lock-free version of evictExpired.
// The caller must hold s.mu.
func (s *Store) evictExpiredLocked(now time.Time) {
	for key, ac := range s.authCodes {
		if now.After(ac.ExpiresAt) {
			delete(s.authCodes, key)
		}
	}

	for key, rt := range s.refreshTokens {
		if now.After(rt.ExpiresAt) {
			delete(s.refreshTokens, key)
		}
	}

	for key, client := range s.clients {
		if now.After(client.CreatedAt.Add(clientTTL)) {
			delete(s.clients, key)
		}
	}
}

// SaveAuthCode stores an authorization code. Returns an error if the maximum number
// of authorization codes has been reached. When the cap is hit, expired codes
// are evicted first before rejecting the request.
func (s *Store) SaveAuthCode(code *Code) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.authCodes) >= maxAuthCodes {
		s.evictExpiredLocked(time.Now())
	}

	if len(s.authCodes) >= maxAuthCodes {
		return errMaxAuthCodesReached
	}

	s.authCodes[code.Code] = code

	return nil
}

// GetAuthCode retrieves an authorization code without deleting it.
// Returns an error if the code is not found or has expired.
func (s *Store) GetAuthCode(code string, now time.Time) (*Code, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ac, ok := s.authCodes[code]
	if !ok {
		return nil, errAuthCodeNotFound
	}

	if now.After(ac.ExpiresAt) {
		return nil, errAuthCodeExpired
	}

	return ac, nil
}

// DeleteAuthCode deletes an authorization code by key.
func (s *Store) DeleteAuthCode(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.authCodes, code)
}

// ConsumeAuthCode retrieves and deletes an authorization code (one-time use).
// Returns an error if the code is not found or has expired.
func (s *Store) ConsumeAuthCode(code string, now time.Time) (*Code, error) {
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

// SaveRefreshToken stores a refresh token. Returns an error if the maximum number
// of refresh tokens has been reached. When the cap is hit, expired tokens
// are evicted first before rejecting the request.
func (s *Store) SaveRefreshToken(token *RefreshToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.refreshTokens) >= maxRefreshTokens {
		s.evictExpiredLocked(time.Now())
	}

	if len(s.refreshTokens) >= maxRefreshTokens {
		return errMaxRefreshTokensReached
	}

	s.refreshTokens[token.Token] = token

	return nil
}

// ConsumeRefreshToken retrieves and deletes a refresh token (rotation).
// The caller is responsible for saving a new refresh token after consuming the old one.
// Returns an error if the token is not found, belongs to a different client, or has expired.
// The clientID check happens before deletion to prevent an attacker from revoking
// another client's refresh token by presenting it with a mismatched client_id.
func (s *Store) ConsumeRefreshToken(token, clientID string, now time.Time) (*RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rt, ok := s.refreshTokens[token]
	if !ok {
		return nil, errRefreshTokenNotFound
	}

	if rt.ClientID != clientID {
		return nil, errRefreshTokenNotFound
	}

	delete(s.refreshTokens, token)

	if now.After(rt.ExpiresAt) {
		return nil, errRefreshTokenNotFound
	}

	return rt, nil
}

// SaveClient stores a registered client. Returns an error if the maximum number
// of registered clients has been reached. When the cap is hit, expired clients
// are evicted first before rejecting the request.
func (s *Store) SaveClient(client *RegisteredClient) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.clients) >= maxClients {
		s.evictExpiredLocked(time.Now())
	}

	if len(s.clients) >= maxClients {
		return errMaxClientsReached
	}

	cp := *client
	cp.RedirectURIs = cloneStrings(client.RedirectURIs)
	cp.GrantTypes = cloneStrings(client.GrantTypes)

	s.clients[cp.ClientID] = &cp

	return nil
}

// GetClient retrieves a registered client by its client ID.
// Returns a deep copy to prevent callers from mutating store data.
// Returns an error if the client is not found.
func (s *Store) GetClient(clientID string) (*RegisteredClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, ok := s.clients[clientID]
	if !ok {
		return nil, errClientNotFound
	}

	cp := *client
	cp.RedirectURIs = cloneStrings(client.RedirectURIs)
	cp.GrantTypes = cloneStrings(client.GrantTypes)

	return &cp, nil
}

func cloneStrings(src []string) []string {
	if src == nil {
		return nil
	}

	dst := make([]string, len(src))
	copy(dst, src)

	return dst
}
