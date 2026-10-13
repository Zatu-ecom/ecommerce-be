package cachekit

import "sync"

// Store names for metric labels and role selection.
const (
	// StoreCache is the volatile role (evictable, fail-open).
	StoreCache = "cache"
	// StoreDurable is the durable role (non-evicting, fail-closed where required).
	StoreDurable = "kv"
)

var (
	defaultCacheMu sync.RWMutex
	defaultCache   Cache
	defaultDurMu   sync.RWMutex
	defaultDurable Durable
)

// SetDefaultCache installs the process-wide volatile cache, wired once at
// boot (main.go) and per test server (SetupTestServer). Strategies read it
// at call time; a nil default disables caching (legacy direct path).
func SetDefaultCache(c Cache) {
	defaultCacheMu.Lock()
	defer defaultCacheMu.Unlock()
	defaultCache = c
}

// DefaultCache returns the process-wide volatile cache, or nil when unwired.
// Callers MUST nil-check: nil means serve without cache, never fail.
func DefaultCache() Cache {
	defaultCacheMu.RLock()
	defer defaultCacheMu.RUnlock()
	return defaultCache
}

// SetDefaultDurable installs the process-wide durable client (version
// counters, and later scheduler-adjacent reads). Nil disables versioned
// lists (strategies fall back to unversioned TTL reads).
func SetDefaultDurable(d Durable) {
	defaultDurMu.Lock()
	defer defaultDurMu.Unlock()
	defaultDurable = d
}

// DefaultDurable returns the process-wide durable client, or nil when unwired.
func DefaultDurable() Durable {
	defaultDurMu.RLock()
	defer defaultDurMu.RUnlock()
	return defaultDurable
}

// CloseDefaults releases the process-wide clients, if set. Wired into
// graceful shutdown alongside the legacy Redis close.
func CloseDefaults() {
	defaultCacheMu.RLock()
	c := defaultCache
	defaultCacheMu.RUnlock()
	if closer, ok := c.(interface{ Close() error }); ok && closer != nil {
		_ = closer.Close()
	}
	defaultDurMu.RLock()
	d := defaultDurable
	defaultDurMu.RUnlock()
	if closer, ok := d.(interface{ Close() error }); ok && closer != nil {
		_ = closer.Close()
	}
}
