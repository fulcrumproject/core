package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// GenerateToken creates a secure random token and sets the TokenHash field
func (a *Agent) GenerateToken() (string, error) {
	// Generate a secure random token (32 bytes = 256 bits)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Convert to base64 for readability
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store only the hash of the token
	a.TokenHash = HashToken(token)

	return token, nil
}

// HashToken creates a secure hash of a token
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// VerifyToken checks if a token matches the stored hash
func (a *Agent) VerifyToken(token string) bool {
	return a.TokenHash == HashToken(token)
}
