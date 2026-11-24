package cachekit

import (
	"context"
	"errors"
	"time"
)

// TokenDenylist stores revoked JWTs on the durable KV role (remaining exp TTL).
type TokenDenylist struct {
	durable Durable
}

var defaultDenylist *TokenDenylist

// SetTokenDenylist installs the process-wide denylist (boot / test server).
func SetTokenDenylist(d *TokenDenylist) {
	defaultDenylist = d
}

// DefaultTokenDenylist returns the wired denylist, or nil when unwired.
func DefaultTokenDenylist() *TokenDenylist {
	return defaultDenylist
}

// NewTokenDenylist builds a denylist. Nil durable fails closed on reads/writes.
func NewTokenDenylist(d Durable) *TokenDenylist {
	if d == nil {
		return nil
	}
	return &TokenDenylist{durable: d}
}

// Revoke records a logged-out token until expiresAt. Durable failures return
// ErrUnavailable.
func (t *TokenDenylist) Revoke(ctx context.Context, tokenString string, expiresAt time.Time) error {
	if t == nil || t.durable == nil {
		return ErrUnavailable
	}
	key, err := DenylistKey(tokenString)
	if err != nil {
		return err
	}
	ttl := DenylistTTL(expiresAt)
	return t.durable.Set(ctx, key, []byte(DenylistValue), ttl)
}

// IsRevoked reports logout revocation. ErrUnavailable means fail closed.
func (t *TokenDenylist) IsRevoked(ctx context.Context, tokenString string) (bool, error) {
	if t == nil || t.durable == nil {
		return false, ErrUnavailable
	}
	key, err := DenylistKey(tokenString)
	if err != nil {
		return false, err
	}
	_, err = t.durable.Get(ctx, key)
	if errors.Is(err, ErrMiss) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
