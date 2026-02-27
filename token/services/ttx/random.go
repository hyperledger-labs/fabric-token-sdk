package ttx

import (
	"github.com/google/uuid"
)

// NonceSize is kept for compatibility but no longer used for direct byte allocation
const (
	NonceSize = 24
)

// GetRandomNonce returns a secure random nonce based on UUID v4
func GetRandomNonce() ([]byte, error) {
	u := uuid.New()
	return u[:], nil
}