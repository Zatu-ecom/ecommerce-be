package cachekit

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// DenylistValue is stored at bl:{sha256(token)} when a session is revoked.
const DenylistValue = "revoked"

// DenylistKey hashes the raw JWT into an allowlisted durable key (bl:{hex64}).
func DenylistKey(token string) (string, error) {
	sum := sha256.Sum256([]byte(token))
	key := "bl:" + hex.EncodeToString(sum[:])
	if err := ValidateKey(key); err != nil {
		return "", err
	}
	return key, nil
}

// DenylistTTL returns remaining JWT lifetime for denylist storage (min 1s).
func DenylistTTL(expiresAt time.Time) time.Duration {
	ttl := time.Until(expiresAt)
	if ttl < time.Second {
		return time.Second
	}
	return ttl
}
