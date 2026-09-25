package config

import (
	"os"
	"strconv"
	"time"
)

// Defaults for the KV roles. Volatile holds restart-losable domain cache
// (fail-open); durable holds scheduler/idempotency/denylist/limiter/version
// data (fail-closed where required). See 012 pre-spec §0.3.
const (
	defaultCacheHost = "cache-volatile"
	defaultCachePort = "6380"
	defaultKVHost    = "cache-durable"
	defaultKVPort    = "6381"

	defaultRedisPoolSize       = 20
	defaultRedisMinIdleConns   = 5
	defaultRedisDialTimeout    = 5 * time.Second
	defaultRedisReadTimeout    = 500 * time.Millisecond
	defaultRedisWriteTimeout   = 500 * time.Millisecond
	defaultRedisPoolTimeout    = 1 * time.Second
	defaultRedisMaxRetries     = 1
	defaultAsyncSETBudget      = 100 * time.Millisecond
	defaultAsyncSETMaxInflight = 64
)

// RedisConfig holds RESP-backend (Redis/Dragonfly) configuration for both
// KV roles. The pre-012 single-instance fields are gone with common/cache:
// every backend access goes through the cachekit role clients.
type RedisConfig struct {
	// Password authenticates the volatile role. Env: CACHE_PASSWORD.
	Password string

	// CacheAddr targets the volatile role (domain cache, fail-open).
	// Env: CACHE_ADDR, else CACHE_HOST:CACHE_PORT.
	CacheAddr string
	// KVAddr targets the durable role (jobs, idempotency, denylist, limiter).
	// Env: KV_ADDR, else KV_HOST:KV_PORT.
	KVAddr string
	// KVPassword authenticates the durable role. Env: KV_PASSWORD,
	// defaults to Password so local/dev needs no extra secret.
	KVPassword string

	// PoolSize caps total connections per client. Env: REDIS_POOL_SIZE.
	PoolSize int
	// MinIdleConns keeps warm connections per client. Env: REDIS_MIN_IDLE_CONNS.
	MinIdleConns int
	// DialTimeout bounds new-connection setup. Env: REDIS_DIAL_TIMEOUT_MS.
	DialTimeout time.Duration
	// ReadTimeout bounds a single read (GET-side budget). Env: REDIS_READ_TIMEOUT_MS.
	ReadTimeout time.Duration
	// WriteTimeout bounds a single write. Env: REDIS_WRITE_TIMEOUT_MS.
	WriteTimeout time.Duration
	// PoolTimeout bounds waiting for a free connection before failing open.
	// Env: REDIS_POOL_TIMEOUT_MS.
	PoolTimeout time.Duration
	// MaxRetries for idempotent commands (GET path fails open instead past this).
	// Env: REDIS_MAX_RETRIES. Retries use jittered backoff in the client.
	MaxRetries int

	// AsyncSETBudget bounds a detached cache-population write.
	// Env: CACHE_ASYNC_SET_BUDGET_MS.
	AsyncSETBudget time.Duration
	// AsyncSETMaxInflight caps concurrent background SETs; overflow drops + counts.
	// Env: CACHE_ASYNC_SET_MAX_INFLIGHT.
	AsyncSETMaxInflight int
}

// loadRedisConfig loads Redis configuration from environment variables.
func loadRedisConfig() RedisConfig {
	password := os.Getenv("CACHE_PASSWORD")
	return RedisConfig{
		Password: password,

		CacheAddr:  cacheAddrOrDefault(),
		KVAddr:     kvAddrOrDefault(),
		KVPassword: getEnvOrDefault("KV_PASSWORD", password),

		PoolSize:            getEnvAsIntOrDefault("REDIS_POOL_SIZE", defaultRedisPoolSize),
		MinIdleConns:        getEnvAsIntOrDefault("REDIS_MIN_IDLE_CONNS", defaultRedisMinIdleConns),
		DialTimeout:         getEnvAsDurationOrDefault("REDIS_DIAL_TIMEOUT_MS", defaultRedisDialTimeout),
		ReadTimeout:         getEnvAsDurationOrDefault("REDIS_READ_TIMEOUT_MS", defaultRedisReadTimeout),
		WriteTimeout:        getEnvAsDurationOrDefault("REDIS_WRITE_TIMEOUT_MS", defaultRedisWriteTimeout),
		PoolTimeout:         getEnvAsDurationOrDefault("REDIS_POOL_TIMEOUT_MS", defaultRedisPoolTimeout),
		MaxRetries:          getEnvAsIntOrDefault("REDIS_MAX_RETRIES", defaultRedisMaxRetries),
		AsyncSETBudget:      getEnvAsDurationOrDefault("CACHE_ASYNC_SET_BUDGET_MS", defaultAsyncSETBudget),
		AsyncSETMaxInflight: getEnvAsIntOrDefault("CACHE_ASYNC_SET_MAX_INFLIGHT", defaultAsyncSETMaxInflight),
	}
}

// cacheAddrOrDefault prefers the explicit CACHE_ADDR, else host:port parts.
func cacheAddrOrDefault() string {
	if addr := os.Getenv("CACHE_ADDR"); addr != "" {
		return addr
	}
	return getEnvOrDefault("CACHE_HOST", defaultCacheHost) + ":" +
		getEnvOrDefault("CACHE_PORT", defaultCachePort)
}

// kvAddrOrDefault prefers the explicit KV_ADDR, else host:port parts.
func kvAddrOrDefault() string {
	if addr := os.Getenv("KV_ADDR"); addr != "" {
		return addr
	}
	return getEnvOrDefault("KV_HOST", defaultKVHost) + ":" +
		getEnvOrDefault("KV_PORT", defaultKVPort)
}

// getEnvAsDurationOrDefault reads an environment variable as milliseconds
// with a duration fallback. Non-numeric or missing values yield the default.
func getEnvAsDurationOrDefault(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if ms, err := strconv.Atoi(val); err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return defaultVal
}

// HasPassword returns true if a password is configured.
func (r *RedisConfig) HasPassword() bool {
	return r.Password != ""
}
