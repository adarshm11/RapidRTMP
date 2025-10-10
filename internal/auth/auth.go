package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"rapidrtmp/pkg/models"
	"sync"
	"time"
)

// Manager handles authentication and authorization
type Manager struct {
	tokens map[string]*models.PublishToken // token -> PublishToken
	mu     sync.RWMutex

	// Config
	defaultExpiration time.Duration
	maxExpiration     time.Duration
}

// New creates a new auth manager
func New() *Manager {
	return &Manager{
		tokens:            make(map[string]*models.PublishToken),
		defaultExpiration: 1 * time.Hour,
		maxExpiration:     24 * time.Hour,
	}
}

// GeneratePublishToken creates a new publish token for a stream
func (m *Manager) GeneratePublishToken(streamKey string, expiresIn int, publisherIP string) (*models.PublishToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	tokenString := hex.EncodeToString(tokenBytes)

	// Calculate expiration
	var expiration time.Duration
	if expiresIn > 0 {
		expiration = time.Duration(expiresIn) * time.Second
	} else {
		expiration = m.defaultExpiration
	}

	// Cap at max expiration
	if expiration > m.maxExpiration {
		expiration = m.maxExpiration
	}

	token := &models.PublishToken{
		Token:       tokenString,
		StreamKey:   streamKey,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(expiration),
		PublisherIP: publisherIP,
		IsUsed:      false,
	}

	m.tokens[tokenString] = token

	// Start cleanup goroutine for this token
	go m.cleanupToken(tokenString, expiration)

	return token, nil
}

// ValidateToken checks if a token is valid for publishing to a stream
func (m *Manager) ValidateToken(tokenString string, streamKey string, publisherIP string) error {
	m.mu.RLock()
	token, exists := m.tokens[tokenString]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("invalid token")
	}

	if !token.IsValid() {
		return fmt.Errorf("token expired or already used")
	}

	if token.StreamKey != streamKey {
		return fmt.Errorf("token not valid for this stream")
	}

	// Optionally validate IP (can be disabled for testing)
	// if token.PublisherIP != publisherIP {
	//     return fmt.Errorf("token not valid for this IP")
	// }

	return nil
}

// MarkTokenUsed marks a token as used
func (m *Manager) MarkTokenUsed(tokenString string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if token, exists := m.tokens[tokenString]; exists {
		token.IsUsed = true
	}
}

// RevokeToken revokes a token
func (m *Manager) RevokeToken(tokenString string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tokens, tokenString)
}

// cleanupToken removes a token after it expires
func (m *Manager) cleanupToken(tokenString string, expiration time.Duration) {
	time.Sleep(expiration + 1*time.Minute) // Wait a bit longer than expiration

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tokens, tokenString)
}

// CleanupExpiredTokens removes all expired tokens (call periodically)
func (m *Manager) CleanupExpiredTokens() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for tokenString, token := range m.tokens {
		if now.After(token.ExpiresAt) {
			delete(m.tokens, tokenString)
		}
	}
}

// GetTokenCount returns the number of active tokens
func (m *Manager) GetTokenCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tokens)
}
