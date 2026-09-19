// Package user — US3 erasure conformance (T044).
//
// Hard account deletion removes every user/seller-keyed entry with exact
// deletes: no prefix globs on the request path, no tombstones retaining the
// data (a tombstone would keep the user's bytes). A sibling seller's keys
// must survive untouched.
package user

import (
	"context"
	"os"
	"testing"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/common/config"
	"ecommerce-be/test/integration/setup"
	usercache "ecommerce-be/user/cache"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// erasureFlags carries the dummy values config validation requires.
var erasureFlags = map[string]string{
	"DB_HOST":    "localhost",
	"DB_PORT":    "5432",
	"DB_USER":    "test",
	"DB_NAME":    "test",
	"REDIS_HOST": "localhost",
	"JWT_SECRET": "erasure-test-secret",
}

// CacheErasureSuite exercises the user-delete purge against the backend
// selected by KV_BACKEND.
type CacheErasureSuite struct {
	suite.Suite
	container *setup.TestContainer
	cache     cachekit.Cache
}

func (s *CacheErasureSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	for k, v := range erasureFlags {
		_ = os.Setenv(k, v)
	}
	config.Reset()
	cfg, err := config.Load()
	require.NoError(s.T(), err)
	s.cache = provider.NewCache(cfg.Redis)
}

func (s *CacheErasureSuite) TearDownSuite() {
	for k := range erasureFlags {
		_ = os.Unsetenv(k)
	}
	config.Reset()
	s.container.Cleanup(s.T())
}

// plant writes one volatile entry synchronously (raw client, no async
// ambiguity — the purge path itself is what is under test).
func (s *CacheErasureSuite) plant(key, value string) {
	s.T().Helper()
	require.NoError(s.T(), s.container.RedisClient.Set(
		context.Background(), key, value, time.Hour).Err())
}

// requireGoneExactly asserts the key is absent (redis miss), not a
// tombstone: deleted-user bytes must not linger in any form.
func (s *CacheErasureSuite) requireGoneExactly(key string) {
	s.T().Helper()
	raw, err := s.container.RedisClient.Get(context.Background(), key).Bytes()
	require.Error(s.T(), err, "key %q must be gone", key)
	require.Empty(s.T(), raw, "key %q must leave no bytes behind", key)
}

// TestUserDelete_PurgesExactKeys plants every §5 user/seller key for seller
// 701 (two users share the seller), purges, and asserts exact removal while
// seller 702's identical shapes survive.
func (s *CacheErasureSuite) TestUserDelete_PurgesExactKeys() {
	ctx := context.Background()

	for _, k := range []string{
		"seller:701:seller:complete",
		"seller:701:currency:default",
		"seller:701:currency:user:901",
		"seller:701:currency:user:902",
		"seller:701:settings",
	} {
		s.plant(k, `{"owner":701}`)
	}
	// Control keys: another seller, same shapes — must survive.
	for _, k := range []string{
		"seller:702:seller:complete",
		"seller:702:settings",
	} {
		s.plant(k, `{"owner":702}`)
	}

	require.NoError(s.T(), usercache.PurgeSellerCache(ctx, s.cache, 701, []uint{901, 902}))

	for _, k := range []string{
		"seller:701:seller:complete",
		"seller:701:currency:default",
		"seller:701:currency:user:901",
		"seller:701:currency:user:902",
		"seller:701:settings",
	} {
		s.requireGoneExactly(k)
	}
	for _, k := range []string{
		"seller:702:seller:complete",
		"seller:702:settings",
	} {
		got, err := s.container.RedisClient.Get(ctx, k).Bytes()
		require.NoError(s.T(), err, "control key %q must survive", k)
		require.JSONEq(s.T(), `{"owner":702}`, string(got))
	}
}

// TestUserDelete_PurgeIsNilSafe proves a nil cache (caching unwired) is a
// no-op, never an error on the delete path.
func (s *CacheErasureSuite) TestUserDelete_PurgeIsNilSafe() {
	require.NoError(s.T(), usercache.PurgeSellerCache(
		context.Background(), nil, 701, []uint{901}))
}

func TestCacheErasureSuite(t *testing.T) {
	suite.Run(t, new(CacheErasureSuite))
}
