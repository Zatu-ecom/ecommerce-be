// Package cachekit_test — US4 durable correctness (T029: conformance T3 + T12).
package cachekit_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/common/config"
	"ecommerce-be/test/integration/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var durableFlags = map[string]string{
	"DB_HOST":    "localhost",
	"DB_PORT":    "5432",
	"DB_USER":    "test",
	"DB_NAME":    "test",
	"REDIS_HOST": "localhost",
	"JWT_SECRET": "durable-correctness-secret",
}

// DurableCorrectnessSuite exercises denylist + Lua limiter on durable KV.
type DurableCorrectnessSuite struct {
	suite.Suite
	container *setup.TestContainer
	durable   cachekit.Durable
	kvAddr    string
}

func (s *DurableCorrectnessSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	for k, v := range durableFlags {
		_ = os.Setenv(k, v)
	}
	config.Reset()
	cfg, err := config.Load()
	require.NoError(s.T(), err)
	s.kvAddr = cfg.Redis.KVAddr
	s.durable = provider.NewDurable(cfg.Redis)
	cachekit.SetTokenDenylist(cachekit.NewTokenDenylist(s.durable))
}

func (s *DurableCorrectnessSuite) TearDownSuite() {
	for k := range durableFlags {
		_ = os.Unsetenv(k)
	}
	config.Reset()
	cachekit.SetTokenDenylist(nil)
	s.container.Cleanup(s.T())
}

// TestT03LuaAtomicity — concurrent INCR+EXPIRE (goroutines + second client).
func (s *DurableCorrectnessSuite) TestT03LuaAtomicity() {
	ctx := context.Background()
	key := "coupon:apply:rate:900001"
	_ = s.durable.Del(ctx, key)

	const workers = 40
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.durable.IncrWithExpire(ctx, key, time.Minute)
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		s.Require().NoError(err)
	}

	n, err := s.durable.IncrWithExpire(ctx, key, time.Minute)
	s.Require().NoError(err)
	s.Equal(int64(workers+1), n)

	ttl, err := s.durable.TTL(ctx, key)
	s.Require().NoError(err)
	s.Greater(ttl, time.Duration(0))
	s.LessOrEqual(ttl, time.Minute)

	// Second client on the same durable backend.
	client2 := provider.New(provider.Options{Addr: s.kvAddr})
	defer client2.Close()
	n2, err := client2.IncrWithExpire(ctx, key, time.Minute)
	s.Require().NoError(err)
	s.Equal(int64(workers+2), n2)
}

// TestT12Denylist — hashed key, remaining-exp TTL, fail-closed when unwired.
func (s *DurableCorrectnessSuite) TestT12Denylist() {
	ctx := context.Background()
	secret := durableFlags["JWT_SECRET"]
	token, err := auth.GenerateToken(auth.TokenUserInfo{
		UserID:    1,
		Email:     "t12@example.com",
		RoleID:    1,
		RoleName:  "customer",
		RoleLevel: 1,
	}, secret)
	s.Require().NoError(err)

	deny := cachekit.NewTokenDenylist(s.durable)
	revoked, err := deny.IsRevoked(ctx, token)
	s.Require().NoError(err)
	s.False(revoked)

	claims, err := auth.ParseToken(token, secret)
	s.Require().NoError(err)
	s.Require().NotNil(claims.ExpiresAt)
	s.Require().NoError(deny.Revoke(ctx, token, claims.ExpiresAt.Time))

	key, err := cachekit.DenylistKey(token)
	s.Require().NoError(err)
	ttl, err := s.durable.TTL(ctx, key)
	s.Require().NoError(err)
	s.Greater(ttl, 30*time.Second)

	revoked, err = deny.IsRevoked(ctx, token)
	s.Require().NoError(err)
	s.True(revoked)

	// Unwired durable → fail closed (logout/auth must not treat as valid).
	unwired := cachekit.NewTokenDenylist(nil)
	_, err = unwired.IsRevoked(ctx, token)
	s.ErrorIs(err, cachekit.ErrUnavailable)
	s.ErrorIs(unwired.Revoke(ctx, token, time.Now().Add(time.Hour)), cachekit.ErrUnavailable)
}

func TestDurableCorrectnessSuite(t *testing.T) {
	suite.Run(t, new(DurableCorrectnessSuite))
}
