// Package cachekit_test — US2 retry-discipline conformance (T058, H4).
//
// Cache GET/SET take exactly one fast attempt and then fail open (GET →
// ErrUnavailable so callers serve the database) or drop and count (volatile
// SET → nil, shed metrics inside). Retrying cache ops adds tail latency for
// zero benefit — the DB fallback IS the retry. Upstream request retries must
// use bounded jittered backoff instead (alerts.md §6).
package cachekit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
)

// deadOptions dials a refusing port with retries disabled (MaxRetries 0 =
// single attempt) and tight budgets so the test proves bounded behavior,
// not backend speed.
func deadOptions() provider.Options {
	return provider.Options{
		Addr:             "127.0.0.1:1",
		ReadTimeout:      200 * time.Millisecond,
		WriteTimeout:     200 * time.Millisecond,
		PoolTimeout:      200 * time.Millisecond,
		MaxRetries:       0,
		AsyncBudget:      50 * time.Millisecond,
		AsyncMaxInflight: 8,
		BreakerThreshold: 1000,
		BreakerCooldown:  time.Minute,
		Recorder:         cachekit.NopRecorder(),
		Role:             cachekit.StoreCache,
	}
}

// TestRetryDiscipline_GetFailsOpenFast proves a dead backend costs one
// bounded attempt: ErrUnavailable (fail-open signal), no retry storm, no
// hang past the read budget.
func TestRetryDiscipline_GetFailsOpenFast(t *testing.T) {
	a := provider.New(deadOptions())
	defer a.Close()

	start := time.Now()
	_, err := a.Get(context.Background(), "seller:7:product:1")
	elapsed := time.Since(start)
	if !errors.Is(err, cachekit.ErrUnavailable) {
		t.Fatalf("dead GET must fail open with ErrUnavailable, got %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("dead GET took %v: retrying cache reads is forbidden (FR-018)", elapsed)
	}
}

// TestRetryDiscipline_SetDropsFast proves volatile population writes never
// block the response and never error: one enqueue, nil return, backend
// failure counted inside (breaker/error metrics), response already served.
func TestRetryDiscipline_SetDropsFast(t *testing.T) {
	a := provider.New(deadOptions())
	defer a.Close()

	start := time.Now()
	err := a.Set(context.Background(), "seller:7:product:1", []byte(`{"id":1}`), time.Minute)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("async volatile SET must never error (drop and count), got %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("volatile SET blocked %v: response must never wait on writes (FR-006)", elapsed)
	}
}

// TestRetryDiscipline_DurableSetFailsClosed proves durable writes stay
// synchronous: failures surface as ErrUnavailable so callers with no DB
// fallback (denylist, SETNX) can fail closed instead of silently dropping.
func TestRetryDiscipline_DurableSetFailsClosed(t *testing.T) {
	opt := deadOptions()
	opt.Role = cachekit.StoreDurable
	a := provider.New(opt)
	defer a.Close()

	start := time.Now()
	err := a.Set(context.Background(), "bl:abc", []byte("revoked"), time.Hour)
	elapsed := time.Since(start)
	if !errors.Is(err, cachekit.ErrUnavailable) {
		t.Fatalf("dead durable SET must fail closed with ErrUnavailable, got %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("durable SET took %v: single bounded attempt only", elapsed)
	}
}
