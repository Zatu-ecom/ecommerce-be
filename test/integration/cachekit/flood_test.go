// Package cachekit_test — US1 load/flood script (T059, M1 / SC-001).
//
// Unique-query flood: one-off random queries cost one cheap marker each —
// memory flat, no big SETs, p99 flat (the §0.4 regression). Repeat-query
// flood: second sightings fill and third sightings hit at p99. Runs against
// the backend selected by KV_BACKEND.
package cachekit_test

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// FloodSuite drives admission-level floods (no flags needed: AdmitList is
// infrastructure, always on).
type FloodSuite struct {
	suite.Suite
	container *setup.TestContainer
	cache     cachekit.Cache
	adapter   *provider.Adapter
}

func (s *FloodSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	c := provider.New(provider.Options{
		Addr: s.container.RedisClient.Options().Addr,
		Role: cachekit.StoreCache,
	})
	ad, ok := any(c).(*provider.Adapter)
	require.True(s.T(), ok)
	s.cache = c
	s.adapter = ad
}

func (s *FloodSuite) TearDownSuite() {
	s.container.Cleanup(s.T())
}

// p99 returns the 99th percentile of samples (SC-001 snappiness budget).
func p99(samples []time.Duration) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	cp := append([]time.Duration(nil), samples...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := (len(cp)*99 + 99) / 100 // ceil(0.99*n)
	if idx < 1 {
		idx = 1
	}
	return cp[idx-1]
}

// usedMemoryBytes reads backend used_memory (INFO memory corroboration).
func (s *FloodSuite) usedMemoryBytes() int64 {
	s.T().Helper()
	info, err := s.container.RedisClient.Info(context.Background(), "memory").Result()
	require.NoError(s.T(), err)
	for _, line := range strings.Split(info, "\r\n") {
		if v, ok := strings.CutPrefix(line, "used_memory:"); ok {
			n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			require.NoError(s.T(), err)
			return n
		}
	}
	s.T().Fatal("INFO memory lacks used_memory on this backend")
	return 0
}

// countSellerKeys counts volatile keys for one seller scope.
func (s *FloodSuite) countSellerKeys(sellerID uint) int {
	s.T().Helper()
	ctx := context.Background()
	total := 0
	var cursor uint64
	for {
		keys, next, err := s.container.RedisClient.Scan(ctx, cursor, fmt.Sprintf("seller:%d:*", sellerID), 500).Result()
		require.NoError(s.T(), err)
		total += len(keys)
		cursor = next
		if cursor == 0 {
			return total
		}
	}
}

// TestFlood_UniqueQueriesLeaveMarkersOnly drives 200 one-off queries:
// every one admitted=false, p99 flat, memory growth bounded to markers,
// zero full page entries.
func (s *FloodSuite) TestFlood_UniqueQueriesLeaveMarkersOnly() {
	ctx := context.Background()
	const sellerID = uint(91)
	const flood = 200
	hashes := make([]string, flood)
	for i := range hashes {
		hashes[i] = fmt.Sprintf("unique-flood-%d-%d", time.Now().UnixNano(), i)
	}

	memBefore := s.usedMemoryBytes()
	var latencies []time.Duration
	for _, h := range hashes {
		start := time.Now()
		admit, err := cachekit.AdmitList(ctx, s.cache, sellerID, h)
		latencies = append(latencies, time.Since(start))
		require.NoError(s.T(), err)
		require.False(s.T(), admit, "one-off query must never earn a full SET")
	}
	require.Less(s.T(), p99(latencies), 100*time.Millisecond,
		"admission p99 must stay flat under unique flood")

	memGrowth := s.usedMemoryBytes() - memBefore
	s.T().Logf("unique flood: %d markers, memory +%d bytes, admit p99 %v", flood, memGrowth, p99(latencies))
	require.Less(s.T(), memGrowth, int64(1<<20), "200 markers must not move memory")

	// Exactly the markers exist — no page entry for any one-off query.
	require.Equal(s.T(), flood, s.countSellerKeys(sellerID),
		"one-off flood must leave markers only")
}

// TestFlood_RepeatQueriesFillAndHit replays the same shapes: second
// sightings admit, fills land, third sightings all hit at p99.
func (s *FloodSuite) TestFlood_RepeatQueriesFillAndHit() {
	ctx := context.Background()
	const sellerID = uint(92)
	const flood = 100
	hashes := make([]string, flood)
	for i := range hashes {
		hashes[i] = fmt.Sprintf("repeat-flood-%d-%d", time.Now().UnixNano(), i)
	}

	for _, h := range hashes {
		admit, err := cachekit.AdmitList(ctx, s.cache, sellerID, h)
		require.NoError(s.T(), err)
		require.False(s.T(), admit)
	}
	var admitLatencies []time.Duration
	for _, h := range hashes {
		start := time.Now()
		admit, err := cachekit.AdmitList(ctx, s.cache, sellerID, h)
		admitLatencies = append(admitLatencies, time.Since(start))
		require.NoError(s.T(), err)
		require.True(s.T(), admit, "repeat query must earn its fill")
		// Repeat sighting stores the page (async bounded).
		require.NoError(s.T(), s.cache.Set(ctx,
			fmt.Sprintf("seller:%d:productlist:v0:%s", sellerID, h),
			[]byte(`{"products":[]}`), 2*time.Minute))
	}
	require.True(s.T(), s.adapter.Flush(10*time.Second), "fills must land")
	s.T().Logf("repeat flood: admit p99 %v", p99(admitLatencies))

	var getLatencies []time.Duration
	hits := 0
	for _, h := range hashes {
		start := time.Now()
		_, err := s.cache.Get(ctx, fmt.Sprintf("seller:%d:productlist:v0:%s", sellerID, h))
		getLatencies = append(getLatencies, time.Since(start))
		if err == nil {
			hits++
		}
	}
	require.Equal(s.T(), flood, hits, "all repeated queries must hit after fill")
	require.Less(s.T(), p99(getLatencies), 100*time.Millisecond,
		"repeat-query p99 must stay flat")
	s.T().Logf("repeat flood: %d/%d hits, get p99 %v", hits, flood, p99(getLatencies))
}

func TestFloodSuite(t *testing.T) {
	suite.Run(t, new(FloodSuite))
}
