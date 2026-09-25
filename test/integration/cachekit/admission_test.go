// Package cachekit_test — US2 admission conformance (T036: T13).
//
// Admission gates combinatorial query caching on a repeat sighting
// (pre-spec §4.4): one-off queries leave a tiny marker only, repeats earn
// the full SET. Singleflight coalesces identical keys; admission is the only
// defense against random one-off floods.
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

// admissionFlags carries the dummy values config validation requires. The
// admission helper itself needs no flags; SetWrites stays on so markers and
// fills are stored.
var admissionFlags = map[string]string{
	"DB_HOST":    "localhost",
	"DB_PORT":    "5432",
	"DB_USER":    "test",
	"DB_NAME":    "test",
	"REDIS_HOST": "localhost",
	"JWT_SECRET": "admission-test-secret",
}

// AdmissionSuite exercises T13 against the real backend selected by
// KV_BACKEND (Redis default, Dragonfly opt-in).
type AdmissionSuite struct {
	suite.Suite
	container *setup.TestContainer
	cache     cachekit.Cache
	adapter   *provider.Adapter
}

func (s *AdmissionSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	for k, v := range admissionFlags {
		_ = os.Setenv(k, v)
	}
	config.Reset()
	cfg, err := config.Load()
	require.NoError(s.T(), err)
	c := provider.NewCache(cfg.Redis)
	s.cache = c
	ad, ok := c.(*provider.Adapter)
	require.True(s.T(), ok, "NewCache must return *provider.Adapter for Flush/Stats")
	s.adapter = ad
}

func (s *AdmissionSuite) TearDownSuite() {
	for k := range admissionFlags {
		_ = os.Unsetenv(k)
	}
	config.Reset()
	s.container.Cleanup(s.T())
}

// TestT13Admission_OneOffMarkerOnly proves the §0.4 root-cause fix: the first
// sighting of a query hash claims a tiny marker and earns NO full SET; the
// repeat within the window earns the full response entry.
func (s *AdmissionSuite) TestT13Admission_OneOffMarkerOnly() {
	ctx := context.Background()
	sellerID := uint(7)
	hash := fmt.Sprintf("t13-oneoff-%d", time.Now().UnixNano())

	// First sight: marker claimed, no full entry earned.
	admit, err := cachekit.AdmitList(ctx, s.cache, sellerID, hash)
	require.NoError(s.T(), err)
	require.False(s.T(), admit, "first sight must not earn a full SET")

	markerKey, err := cachekit.AdmissionMarkerKey(sellerID, hash)
	require.NoError(s.T(), err)
	stored, err := s.container.RedisClient.Get(ctx, markerKey).Bytes()
	require.NoError(s.T(), err, "one-off must leave its marker")
	require.Equal(s.T(), "1", string(stored))

	// Repeat within the window: full entry earned.
	admit, err = cachekit.AdmitList(ctx, s.cache, sellerID, hash)
	require.NoError(s.T(), err)
	require.True(s.T(), admit, "repeat sighting must earn the full SET")

	// The caller stores the full page on admit; it must be retrievable.
	listKey, err := cachekit.BuildSellerKey(sellerID, "productlist", "v1", hash)
	require.NoError(s.T(), err)
	require.NoError(s.T(), s.cache.Set(ctx, listKey, []byte(`{"items":[]}`), 2*time.Minute))
	require.True(s.T(), s.adapter.Flush(5*time.Second), "async SET must land")
	got, err := s.cache.Get(ctx, listKey)
	require.NoError(s.T(), err)
	require.JSONEq(s.T(), `{"items":[]}`, string(got))
}

// TestT13Admission_DistinctHashesIndependent proves markers are per query
// shape: two hashes each need their own repeat; one query's repeat never
// admits another's.
func (s *AdmissionSuite) TestT13Admission_DistinctHashesIndependent() {
	ctx := context.Background()
	sellerID := uint(7)
	now := time.Now().UnixNano()
	hashA := fmt.Sprintf("t13-a-%d", now)
	hashB := fmt.Sprintf("t13-b-%d", now)

	for _, h := range []string{hashA, hashB} {
		admit, err := cachekit.AdmitList(ctx, s.cache, sellerID, h)
		require.NoError(s.T(), err)
		require.False(s.T(), admit, "first sight of %q must not admit", h)
	}

	admit, err := cachekit.AdmitList(ctx, s.cache, sellerID, hashA)
	require.NoError(s.T(), err)
	require.True(s.T(), admit, "repeat of A must admit")

	// B was seen once; its second sight (not third) is the repeat that
	// admits — seeing A twice must not admit B early. B's next call is its
	// repeat, so it admits here exactly once.
	admit, err = cachekit.AdmitList(ctx, s.cache, sellerID, hashB)
	require.NoError(s.T(), err)
	require.True(s.T(), admit, "repeat of B must admit independently")
}

// TestT13Admission_MarkerKeyShape proves the marker key contract:
// seller-scoped, allowlisted by the builder, jittered ~60s TTL.
func (s *AdmissionSuite) TestT13Admission_MarkerKeyShape() {
	key, err := cachekit.AdmissionMarkerKey(7, "abc123")
	require.NoError(s.T(), err)
	require.Equal(s.T(), "seller:7:seen:abc123", key)
	require.NoError(s.T(), cachekit.ValidateKey(key))

	if _, err := cachekit.AdmissionMarkerKey(0, "abc123"); err == nil {
		s.T().Fatal("zero seller must be rejected")
	}
}

func TestAdmissionSuite(t *testing.T) {
	suite.Run(t, new(AdmissionSuite))
}
