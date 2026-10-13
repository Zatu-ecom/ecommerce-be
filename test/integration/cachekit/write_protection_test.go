// Package cachekit_test — US2 write-protection conformance (T038).
//
// Covers the §0.4 chain-breakers that are not admission: the write-shed
// breaker (T15), jittered wire TTLs (T16), and generation-guarded SETs that
// cannot resurrect stale data after a concurrent write (T17).
package cachekit_test

import (
	"context"
	"fmt"
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

// writeProtectionFlags carries the dummy values config validation requires.
var writeProtectionFlags = map[string]string{
	"DB_HOST":    "localhost",
	"DB_PORT":    "5432",
	"DB_USER":    "test",
	"DB_NAME":    "test",
	"REDIS_HOST": "localhost",
	"JWT_SECRET": "write-protection-secret",
}

// WriteProtectionSuite exercises T15/T16/T17 against the backend selected by
// KV_BACKEND.
type WriteProtectionSuite struct {
	suite.Suite
	container *setup.TestContainer
	cache     cachekit.Cache
	adapter   *provider.Adapter
}

func (s *WriteProtectionSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	for k, v := range writeProtectionFlags {
		_ = os.Setenv(k, v)
	}
	config.Reset()
	cfg, err := config.Load()
	require.NoError(s.T(), err)
	c := provider.NewCache(cfg.Redis)
	s.cache = c
	ad, ok := c.(*provider.Adapter)
	require.True(s.T(), ok, "NewCache must return *provider.Adapter")
	ad.SetBreakerThreshold(1000)
	ad.SetBreakerCooldown(time.Minute)
	s.adapter = ad
}

func (s *WriteProtectionSuite) TearDownSuite() {
	for k := range writeProtectionFlags {
		_ = os.Unsetenv(k)
	}
	config.Reset()
	s.container.Cleanup(s.T())
}

// TestT15WriteShed_TripAndRecover proves the automatic last resort
// (pre-spec §8.6.5): a sustained SET error storm pauses domain writes while
// GETs continue, and writes resume automatically after the cooldown.
func (s *WriteProtectionSuite) TestT15WriteShed_TripAndRecover() {
	ctx := context.Background()
	dead := provider.New(provider.Options{
		Addr:             "127.0.0.1:1",
		ReadTimeout:      200 * time.Millisecond,
		WriteTimeout:     200 * time.Millisecond,
		PoolTimeout:      200 * time.Millisecond,
		MaxRetries:       0,
		AsyncBudget:      50 * time.Millisecond,
		AsyncMaxInflight: 16,
		BreakerThreshold: 3,
		BreakerCooldown:  time.Second,
		Recorder:         cachekit.NopRecorder(),
		Role:             cachekit.StoreCache,
	})
	defer dead.Close()

	// Error storm: background SETs fail until the breaker trips.
	for i := 0; i < 6; i++ {
		require.NoError(s.T(), dead.Set(ctx, "seller:7:product:t15", []byte(`{}`), time.Minute))
	}
	tripped := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if dead.Stats().ShedActive {
			tripped = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.True(s.T(), tripped, "error storm must trip the breaker, stats=%+v", dead.Stats())

	// Reads continue while writes shed: GETs fail open fast, never hang.
	start := time.Now()
	_, err := dead.Get(ctx, "seller:7:product:t15")
	require.Error(s.T(), err)
	require.Less(s.T(), time.Since(start), 2*time.Second, "GETs must continue while shed")

	// Shed SETs skip immediately and count.
	before := dead.Stats().ShedSkipped
	require.NoError(s.T(), dead.Set(ctx, "seller:7:product:t15", []byte(`{}`), time.Minute))
	require.Greater(s.T(), dead.Stats().ShedSkipped, before, "shed writes must count")

	// Cooldown expiry recovers automatically (flag clears; the next probe
	// re-trips only if the backend is still sick).
	time.Sleep(1200 * time.Millisecond)
	require.False(s.T(), dead.ShedActive(), "breaker must recover after cooldown")
}

// TestT15WriteShed_ManualLever proves CACHE_SET_WRITES=off serves GETs while
// skipping all domain SETs (the manual form of the §0.4 intervention).
func (s *WriteProtectionSuite) TestT15WriteShed_ManualLever() {
	ctx := context.Background()
	key := fmt.Sprintf("seller:7:product:t15manual-%d", time.Now().UnixNano())
	_ = s.cache.Del(ctx, key)

	s.adapter.SetSetWritesEnabled(false)
	defer s.adapter.ClearSetWritesOverride()
	require.True(s.T(), s.adapter.ShedActive(), "manual off must report shed-active")

	before := s.adapter.Stats().ShedSkipped
	require.NoError(s.T(), s.cache.Set(ctx, key, []byte(`{"id":1}`), time.Minute))
	s.adapter.Flush(2 * time.Second)
	require.Greater(s.T(), s.adapter.Stats().ShedSkipped, before, "manual-off SETs must count")

	_, err := s.cache.Get(ctx, key)
	require.Error(s.T(), err, "manual-off SET must store nothing")

	s.adapter.ClearSetWritesOverride()
	require.False(s.T(), s.adapter.ShedActive(), "manual lever must release")
	require.NoError(s.T(), s.cache.Set(ctx, key, []byte(`{"id":1}`), time.Minute))
	require.True(s.T(), s.adapter.Flush(5*time.Second))
	got, err := s.cache.Get(ctx, key)
	require.NoError(s.T(), err)
	require.JSONEq(s.T(), `{"id":1}`, string(got))
}

// TestT16Jitter_WireTTLs proves every wire TTL falls inside base ±15%
// (pre-spec §5.6): batches written together never expire together.
func (s *WriteProtectionSuite) TestT16Jitter_WireTTLs() {
	ctx := context.Background()
	const base = 60 * time.Second
	lo := 51 * time.Second // 60 * 0.85
	hi := 69 * time.Second // 60 * 1.15

	// Pure helper bound across a batch.
	for i := 0; i < 50; i++ {
		got := cachekit.JitteredTTL(base)
		require.GreaterOrEqual(s.T(), got, lo, "jitter below -15%%")
		require.LessOrEqual(s.T(), got, hi, "jitter above +15%%")
		require.Positive(s.T(), int64(got))
	}

	// Wire bound: TTLs read back from the backend sit in the same window
	// (2s slack covers suite time between SET and TTL).
	now := time.Now().UnixNano()
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("seller:7:product:t16-%d-%d", now, i)
		require.NoError(s.T(), s.cache.Set(ctx, key, []byte(`{"id":1}`), cachekit.JitteredTTL(base)))
	}
	require.True(s.T(), s.adapter.Flush(5*time.Second), "async SETs must land before TTL reads")
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("seller:7:product:t16-%d-%d", now, i)
		ttl, err := s.container.RedisClient.TTL(ctx, key).Result()
		require.NoError(s.T(), err)
		require.Greater(s.T(), ttl, 45*time.Second, "wire TTL %v below bound for %q", ttl, key)
		require.LessOrEqual(s.T(), ttl, hi, "wire TTL %v above bound for %q", ttl, key)
	}
}

// TestT17GenerationGuard_NoResurrection proves the §0.2 race guard: a fill
// that missed before a concurrent write Del must drop its stale SET instead
// of resurrecting pre-write bytes.
func (s *WriteProtectionSuite) TestT17GenerationGuard_NoResurrection() {
	ctx := context.Background()
	dataKey := fmt.Sprintf("seller:7:product:t17-%d", time.Now().UnixNano())
	genKey := cachekit.GenerationKeyFor(dataKey)

	// Miss-time capture: the filler saw generation 1.
	gen1, err := s.cache.Incr(ctx, genKey)
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint64(1), gen1)

	// Concurrent write commits: Del + generation bump (the write path).
	require.NoError(s.T(), s.cache.Del(ctx, dataKey))
	gen2, err := s.cache.Incr(ctx, genKey)
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint64(2), gen2)

	// Stale fill with the pre-write generation must drop (KEYS-only Lua).
	stored, err := s.cache.CompareAndSet(ctx, dataKey, genKey, gen1, []byte(`{"stale":true}`), time.Minute)
	require.NoError(s.T(), err)
	require.False(s.T(), stored, "stale generation must not store")
	require.GreaterOrEqual(s.T(), s.adapter.Stats().GenerationDropped, int64(1),
		"generation drops must count as cache_set_generation_dropped")

	_, err = s.cache.Get(ctx, dataKey)
	require.Error(s.T(), err, "dropped SET must leave no resurrected entry")

	// Fresh fill with the current generation stores.
	stored, err = s.cache.CompareAndSet(ctx, dataKey, genKey, gen2, []byte(`{"fresh":true}`), time.Minute)
	require.NoError(s.T(), err)
	require.True(s.T(), stored, "current generation must store")
	got, err := s.cache.Get(ctx, dataKey)
	require.NoError(s.T(), err)
	require.JSONEq(s.T(), `{"fresh":true}`, string(got))

	// Missing marker fails safe: never stores into an unknown generation.
	stored, err = s.cache.CompareAndSet(ctx,
		fmt.Sprintf("seller:7:product:t17-missing-%d", time.Now().UnixNano()),
		fmt.Sprintf("seller:7:product:t17-missing-%d:gen", time.Now().UnixNano()),
		99, []byte(`{}`), time.Minute)
	require.NoError(s.T(), err)
	require.False(s.T(), stored, "unknown generation must fail safe")
}

func TestWriteProtectionSuite(t *testing.T) {
	suite.Run(t, new(WriteProtectionSuite))
}
