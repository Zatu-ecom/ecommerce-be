// Construction entry points for the cachekit backends live in the provider
// package so the dependency direction stays one-way (provider -> cachekit),
// exactly like database/sql drivers. Wiring sites (factories, main) import
// this package ONLY to call NewCache/NewDurable once; provider types must
// never appear in any other signature, struct, or import.
package provider

import (
	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/config"
)

// NewCache builds the volatile-role Cache from CACHE_ADDR / pool settings.
// A failing backend does NOT fail construction: domain reads fail open to
// the database (pre-spec §8.3). Callers may Ping to observe health.
func NewCache(cfg config.RedisConfig) cachekit.Cache {
	return New(Options{
		Addr:         cfg.CacheAddr,
		Password:     cfg.Password,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
		MaxRetries:   cfg.MaxRetries,
	})
}

// NewDurable builds the durable-role client (Durable + DelayQueue) from
// KV_ADDR / pool settings. Callers SHOULD Ping before starting queue
// consumers: without durable KV the scheduler must not run.
func NewDurable(cfg config.RedisConfig) cachekit.DurableQueue {
	return New(Options{
		Addr:         cfg.KVAddr,
		Password:     cfg.KVPassword,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
		MaxRetries:   cfg.MaxRetries,
	})
}
