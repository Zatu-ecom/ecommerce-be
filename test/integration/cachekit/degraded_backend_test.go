// Package cachekit_test — US2 degraded-backend conformance (T037: T14).
//
// Slow/dead volatile backend: domain reads still succeed via the database
// with bounded extra latency, responses never wait for cache writes, and
// skipped writes are dropped and counted (pre-spec §8.6, FR-002/FR-006).
package cachekit_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/common/config"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// degradedFlags carries the dummy values config validation requires.
var degradedFlags = map[string]string{
	"DB_HOST":    "localhost",
	"DB_PORT":    "5432",
	"DB_USER":    "test",
	"DB_NAME":    "test",
	"REDIS_HOST": "localhost",
	"JWT_SECRET": "degraded-test-secret",
}

// DegradedBackendSuite exercises T14. The healthy container backend proves
// the baseline; a dead-end Adapter (closed port) proves the degraded path.
type DegradedBackendSuite struct {
	suite.Suite
	container *setup.TestContainer
}

func (s *DegradedBackendSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	for k, v := range degradedFlags {
		_ = os.Setenv(k, v)
	}
	config.Reset()
	_, err := config.Load()
	require.NoError(s.T(), err)
}

func (s *DegradedBackendSuite) TearDownSuite() {
	for k := range degradedFlags {
		_ = os.Unsetenv(k)
	}
	config.Reset()
	s.container.Cleanup(s.T())
}

// deadAdapter dials a port that answers nothing (connection refused): the
// fastest deterministic "dead backend". Short timeouts keep the suite fast;
// the async budget bounds population writes.
func (s *DegradedBackendSuite) deadAdapter() *provider.Adapter {
	return provider.New(provider.Options{
		Addr:             "127.0.0.1:1",
		ReadTimeout:      200 * time.Millisecond,
		WriteTimeout:     200 * time.Millisecond,
		PoolTimeout:      200 * time.Millisecond,
		MaxRetries:       0,
		AsyncBudget:      50 * time.Millisecond,
		AsyncMaxInflight: 16,
		BreakerThreshold: 1000,
		BreakerCooldown:  time.Minute,
		Recorder:         cachekit.NopRecorder(),
		Role:             cachekit.StoreCache,
	})
}

// TestT14DegradedBackend_ReadsFailOpen proves domain reads degrade to the
// database path with bounded extra latency when volatile is dead: GETs
// surface ErrUnavailable fast (never a timeout cascade), and the fill path
// still serves the database bytes.
func (s *DegradedBackendSuite) TestT14DegradedBackend_ReadsFailOpen() {
	ctx := context.Background()
	dead := s.deadAdapter()
	defer dead.Close()

	start := time.Now()
	_, err := dead.Get(ctx, "seller:7:product:1")
	elapsed := time.Since(start)
	require.Error(s.T(), err)
	require.True(s.T(), errors.Is(err, cachekit.ErrUnavailable), "dead GET must fail open, got %v", err)
	require.Less(s.T(), elapsed, 2*time.Second, "dead GET must not hang past its budget")

	// Fill path: miss/unavailable → singleflight DB load → return bytes while
	// the async SET drops in the background (never blocking the response).
	fillStart := time.Now()
	body, err := cachekit.FetchBytes(ctx, dead, cachekit.NopRecorder(), &cachekit.Flight{},
		"product", "detail", "seller:7:product:1", time.Minute, time.Minute,
		func(context.Context) ([]byte, error) { return []byte(`{"id":1}`), nil })
	require.NoError(s.T(), err)
	require.JSONEq(s.T(), `{"id":1}`, string(body))
	require.Less(s.T(), time.Since(fillStart), 2*time.Second, "fill must not wait for dead SETs")
}

// TestT14DegradedBackend_SetsDropAndCount proves population writes on a dead
// backend never slow the response: Set returns immediately and the shed/drop
// counters record the skipped work (breaker threshold is high here so drops
// are pool/background failures, counted via shed+async metrics after trip).
func (s *DegradedBackendSuite) TestT14DegradedBackend_SetsDropAndCount() {
	ctx := context.Background()
	dead := provider.New(provider.Options{
		Addr:             "127.0.0.1:1",
		ReadTimeout:      200 * time.Millisecond,
		WriteTimeout:     200 * time.Millisecond,
		PoolTimeout:      200 * time.Millisecond,
		MaxRetries:       0,
		AsyncBudget:      50 * time.Millisecond,
		AsyncMaxInflight: 16,
		BreakerThreshold: 2,
		BreakerCooldown:  300 * time.Millisecond,
		Recorder:         cachekit.NopRecorder(),
		Role:             cachekit.StoreCache,
	})
	defer dead.Close()

	// Response path never waits: every Set returns nil immediately.
	for i := 0; i < 5; i++ {
		start := time.Now()
		require.NoError(s.T(), dead.Set(ctx, "seller:7:product:1", []byte(`{"id":1}`), time.Minute))
		require.Less(s.T(), time.Since(start), 500*time.Millisecond, "async Set must not block")
	}

	// Background failures feed the breaker until writes shed and count.
	deadline := time.Now().Add(5 * time.Second)
	for {
		st := dead.Stats()
		if st.ShedActive || st.ShedSkipped > 0 {
			break
		}
		if time.Now().After(deadline) {
			s.T().Fatalf("dead-backend SETs must shed and count, stats=%+v", st)
		}
		time.Sleep(50 * time.Millisecond)
	}
	st := dead.Stats()
	s.T().Logf("degraded stats: %+v", st)
	require.True(s.T(), st.ShedActive || st.ShedSkipped > 0 || st.AsyncDropped > 0,
		"skipped writes must be counted, stats=%+v", st)
}

// TestT14DegradedBackend_HealthyBaseline bounds the test itself: the same
// read against the healthy container backend serves from cache after one
// async fill (proves the suite can tell healthy from degraded).
func (s *DegradedBackendSuite) TestT14DegradedBackend_HealthyBaseline() {
	ctx := context.Background()
	cfg := config.Get()
	require.NotNil(s.T(), cfg)
	healthy := provider.NewCache(cfg.Redis)
	ad, ok := healthy.(*provider.Adapter)
	require.True(s.T(), ok)
	defer ad.Close()

	key := "seller:7:product:t14baseline"
	_ = healthy.Del(ctx, key)
	body, err := cachekit.FetchBytes(ctx, healthy, cachekit.NopRecorder(), &cachekit.Flight{},
		"product", "detail", key, time.Minute, time.Minute,
		func(context.Context) ([]byte, error) { return []byte(`{"id":2}`), nil })
	require.NoError(s.T(), err)
	require.JSONEq(s.T(), `{"id":2}`, string(body))
	require.True(s.T(), ad.Flush(5*time.Second), "healthy async SET must land")

	got, err := healthy.Get(ctx, key)
	require.NoError(s.T(), err, "healthy backend must serve the filled entry")
	require.JSONEq(s.T(), `{"id":2}`, string(got))
}

func TestDegradedBackendSuite(t *testing.T) {
	suite.Run(t, new(DegradedBackendSuite))
}
