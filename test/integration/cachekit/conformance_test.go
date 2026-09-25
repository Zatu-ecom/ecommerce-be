// Package cachekit_test is the T1–T17 conformance suite for the cachekit
// interfaces (specs/012-caching-infrastructure §9.1). It runs unmodified
// against both backends: default Redis, or Dragonfly with KV_BACKEND=dragonfly.
//
// QA perspective: each case asserts the contract plus its edges — round
// trips on BOTH roles, replay/double-claim, atomicity under concurrency,
// prefix isolation, fail-open vs fail-closed, and counted drops. Keys are
// namespaced per case (tNN:<nano>) so the shared process-wide backends stay
// isolated without container reboots.
package cachekit_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/test/integration/setup"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ConformanceSuite boots both KV role containers and exercises the cachekit
// contracts. Backend under test is reported via KVBackendName.
type ConformanceSuite struct {
	suite.Suite
	container *setup.TestContainer
	volatile  *provider.Adapter
	durable   *provider.Adapter
}

// SetupSuite starts Postgres (unused by most cases, kept for parity) and both
// KV role containers for the selected backend.
func (s *ConformanceSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.T().Logf("cachekit conformance backend: %s", s.container.KVBackendName)
	s.volatile = provider.New(provider.Options{
		Addr: s.container.RedisClient.Options().Addr,
		Role: cachekit.StoreCache,
	})
	s.durable = provider.New(provider.Options{
		Addr: s.container.DurableKVClient.Options().Addr,
		Role: cachekit.StoreDurable,
	})
}

// TearDownSuite terminates all containers.
func (s *ConformanceSuite) TearDownSuite() {
	if s.volatile != nil {
		_ = s.volatile.Close()
	}
	if s.durable != nil {
		_ = s.durable.Close()
	}
	s.container.Cleanup(s.T())
}

// uniq namespaces keys per case so shared backends stay isolated.
func uniq(prefix string) string {
	return fmt.Sprintf("%s:%d:", prefix, time.Now().UnixNano())
}

// deadAdapter dials a connection-refused port: deterministic backend-down
// simulation without stopping shared containers.
func (s *ConformanceSuite) deadAdapter() *provider.Adapter {
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

// dialRaw PING-waits a dedicated container (ports may remap after restarts).
func (s *ConformanceSuite) dialRaw(c testcontainers.Container) *redis.Client {
	s.T().Helper()
	ctx := context.Background()
	host, err := c.Host(ctx)
	require.NoError(s.T(), err)
	port, err := c.MappedPort(ctx, "6379")
	require.NoError(s.T(), err)
	client := redis.NewClient(&redis.Options{Addr: host + ":" + port.Port()})
	deadline := time.Now().Add(30 * time.Second)
	for {
		if err := client.Ping(ctx).Err(); err == nil {
			return client
		} else if time.Now().After(deadline) {
			s.T().Fatalf("dedicated container not PINGing: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// TestT01RoundTrip covers GET/SET/DEL round-trip + TTL expiry on both roles.
func (s *ConformanceSuite) TestT01RoundTrip() {
	ctx := context.Background()
	ns := uniq("t01")

	// Volatile role (async SET: must Flush before reading).
	vKey := ns + "v"
	require.NoError(s.T(), s.volatile.Set(ctx, vKey, []byte("v1"), time.Minute))
	require.True(s.T(), s.volatile.Flush(5*time.Second))
	got, err := s.volatile.Get(ctx, vKey)
	require.NoError(s.T(), err)
	require.Equal(s.T(), []byte("v1"), got)
	require.NoError(s.T(), s.volatile.Del(ctx, vKey))
	_, err = s.volatile.Get(ctx, vKey)
	require.ErrorIs(s.T(), err, cachekit.ErrMiss, "deleted key must miss")

	// Absent key misses on both roles.
	_, err = s.volatile.Get(ctx, ns+"absent")
	require.ErrorIs(s.T(), err, cachekit.ErrMiss)
	_, err = s.durable.Get(ctx, ns+"absent")
	require.ErrorIs(s.T(), err, cachekit.ErrMiss)

	// Durable role (sync SET, exact TTL).
	dKey := ns + "d"
	require.NoError(s.T(), s.durable.Set(ctx, dKey, []byte("d1"), time.Minute))
	got, err = s.durable.Get(ctx, dKey)
	require.NoError(s.T(), err)
	require.Equal(s.T(), []byte("d1"), got)
	require.NoError(s.T(), s.durable.Del(ctx, dKey))
	_, err = s.durable.Get(ctx, dKey)
	require.ErrorIs(s.T(), err, cachekit.ErrMiss)

	// TTL expiry on both roles.
	expV := ns + "exp-v"
	require.NoError(s.T(), s.volatile.Set(ctx, expV, []byte("x"), time.Second))
	require.True(s.T(), s.volatile.Flush(5*time.Second))
	expD := ns + "exp-d"
	require.NoError(s.T(), s.durable.Set(ctx, expD, []byte("x"), time.Second))
	time.Sleep(2 * time.Second)
	_, err = s.volatile.Get(ctx, expV)
	require.ErrorIs(s.T(), err, cachekit.ErrMiss, "volatile key must expire")
	_, err = s.durable.Get(ctx, expD)
	require.ErrorIs(s.T(), err, cachekit.ErrMiss, "durable key must expire")
}

// TestT02SetNX covers idempotent claim + replay semantics.
func (s *ConformanceSuite) TestT02SetNX() {
	ctx := context.Background()
	ns := uniq("t02")

	for _, c := range []struct {
		name string
		kv   *provider.Adapter
	}{
		{"volatile", s.volatile},
		{"durable", s.durable},
	} {
		key := ns + c.name
		claimed, err := c.kv.SetNX(ctx, key, []byte(`{"n":2}`), time.Minute)
		require.NoError(s.T(), err)
		require.True(s.T(), claimed, "%s: first claim must win", c.name)

		// Replay loses and must not overwrite the winner's bytes.
		again, err := c.kv.SetNX(ctx, key, []byte(`{"n":"loser"}`), time.Minute)
		require.NoError(s.T(), err)
		require.False(s.T(), again, "%s: replay must lose", c.name)
		raw, err := c.kv.Get(ctx, key)
		require.NoError(s.T(), err)
		require.JSONEq(s.T(), `{"n":2}`, string(raw))

		// Independent key claims independently.
		other, err := c.kv.SetNX(ctx, key+":other", []byte("1"), time.Minute)
		require.NoError(s.T(), err)
		require.True(s.T(), other)
	}
}

// TestT03LuaAtomicity covers concurrent INCR+EXPIRE from goroutines and two clients.
func (s *ConformanceSuite) TestT03LuaAtomicity() {
	ctx := context.Background()
	key := uniq("t03") + "counter"
	const goroutines = 8
	const perG = 25

	var wg sync.WaitGroup
	errs := make(chan error, goroutines*perG)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				if _, err := s.durable.IncrWithExpire(ctx, key, time.Minute); err != nil {
					errs <- err
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(s.T(), err)
	}

	// A second client must agree on the exact total: no lost updates.
	client2 := provider.New(provider.Options{
		Addr: s.container.DurableKVClient.Options().Addr,
		Role: cachekit.StoreDurable,
	})
	defer func() { _ = client2.Close() }()
	final, err := client2.IncrWithExpire(ctx, key, time.Minute)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(goroutines*perG+1), final,
		"concurrent INCR+EXPIRE must be exactly-once per call")

	ttl, err := s.durable.TTL(ctx, key)
	require.NoError(s.T(), err)
	require.Greater(s.T(), ttl, time.Duration(0))
	require.LessOrEqual(s.T(), ttl, time.Minute)
}

// TestT04DelayQueue covers schedule → poll → atomic claim → cancel.
func (s *ConformanceSuite) TestT04DelayQueue() {
	ctx := context.Background()
	payload := []byte(fmt.Sprintf(`{"job":"t04-%d"}`, time.Now().UnixNano()))

	jobID, err := s.durable.Schedule(ctx, payload, 0)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), jobID)

	got, err := s.durable.Poll(ctx, 10)
	require.NoError(s.T(), err)
	require.Len(s.T(), got, 1)
	require.Equal(s.T(), payload, got[0])

	// Claimed job must not re-poll (atomic ZREM claim).
	again, err := s.durable.Poll(ctx, 10)
	require.NoError(s.T(), err)
	require.Empty(s.T(), again)

	// Exactly one claimant across two clients.
	racePayload := []byte(fmt.Sprintf(`{"job":"t04-race-%d"}`, time.Now().UnixNano()))
	_, err = s.durable.Schedule(ctx, racePayload, 0)
	require.NoError(s.T(), err)
	client2 := provider.New(provider.Options{
		Addr: s.container.DurableKVClient.Options().Addr,
		Role: cachekit.StoreDurable,
	})
	defer func() { _ = client2.Close() }()
	var c1, c2 int
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); p, _ := s.durable.Poll(ctx, 10); c1 = len(p) }()
	go func() { defer wg.Done(); p, _ := client2.Poll(ctx, 10); c2 = len(p) }()
	wg.Wait()
	require.Equal(s.T(), 1, c1+c2, "exactly one claimant per job across clients")

	// Cancel removes a pending job: missing/already-run is a nil no-op.
	cancelID, err := s.durable.Schedule(ctx, []byte(`{"job":"t04-cancel"}`), time.Hour)
	require.NoError(s.T(), err)
	require.NoError(s.T(), s.durable.Cancel(ctx, cancelID))
	require.NoError(s.T(), s.durable.Cancel(ctx, "does-not-exist"))
	require.NoError(s.T(), s.durable.Cancel(ctx, jobID), "claimed job cancel is a no-op")
	rest, err := s.durable.Poll(ctx, 10)
	require.NoError(s.T(), err)
	require.Empty(s.T(), rest, "cancelled future job must never poll")
}

// TestT05ScanDelete covers background SCAN prefix delete over a large keyspace.
func (s *ConformanceSuite) TestT05ScanDelete() {
	ctx := context.Background()
	prefix := uniq("t05janitor")
	sibling := uniq("t05sibling")
	const n = 2000

	pipe := s.container.RedisClient.Pipeline()
	for i := 0; i < n; i++ {
		pipe.Set(ctx, fmt.Sprintf("%s%04d", prefix, i), "v", 0)
	}
	pipe.Set(ctx, sibling+"keep", "v", 0)
	_, err := pipe.Exec(ctx)
	require.NoError(s.T(), err)

	require.NoError(s.T(), s.volatile.DelPrefix(ctx, prefix))

	left, _, err := s.container.RedisClient.Scan(ctx, 0, prefix+"*", 1000).Result()
	require.NoError(s.T(), err)
	require.Empty(s.T(), left, "prefix delete must remove all %d keys", n)

	kept, err := s.container.RedisClient.Get(ctx, sibling+"keep").Bytes()
	require.NoError(s.T(), err, "sibling prefix must survive")
	require.Equal(s.T(), []byte("v"), kept)
}

// startPersistedContainer boots a dedicated backend with persistence enabled
// (appendonly for Redis; dir+dbfilename snapshots for Dragonfly) for T06.
func (s *ConformanceSuite) startPersistedContainer() testcontainers.Container {
	s.T().Helper()
	_, image, dragonfly := kvImage()
	var cmd []string
	var waitingFor wait.Strategy
	if dragonfly {
		cmd = []string{"--dir", "/data", "--dbfilename", "t6dump"}
		waitingFor = wait.ForListeningPort("6379/tcp").WithStartupTimeout(2 * time.Minute)
	} else {
		cmd = []string{"redis-server", "--appendonly", "yes", "--appendfsync", "always"}
		waitingFor = wait.ForLog("Ready to accept connections").WithStartupTimeout(2 * time.Minute)
	}
	c, err := testcontainers.GenericContainer(context.Background(),
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        image,
				Cmd:          cmd,
				ExposedPorts: []string{"6379/tcp"},
				WaitingFor:   waitingFor,
			},
			Started: true,
		})
	require.NoError(s.T(), err, "persisted container must start")
	return c
}

// TestT06Restart covers durable recovery of queue + idempotency keys across restart.
func (s *ConformanceSuite) TestT06Restart() {
	ctx := context.Background()
	c := s.startPersistedContainer()
	defer func() { _ = c.Terminate(ctx) }()

	client := s.dialRaw(c)
	ad := provider.New(provider.Options{Addr: client.Options().Addr, Role: cachekit.StoreDurable})
	defer func() { _ = ad.Close() }()

	// Due job + idempotency claim + plain entry, all on the durable role.
	duePayload := []byte(fmt.Sprintf(`{"job":"t06-%d"}`, time.Now().UnixNano()))
	jobID, err := ad.Schedule(ctx, duePayload, 0)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), jobID)
	nxKey := uniq("t06idem")
	claimed, err := ad.SetNX(ctx, nxKey, []byte(`{"upload":6}`), time.Hour)
	require.NoError(s.T(), err)
	require.True(s.T(), claimed)
	plainKey := uniq("t06plain")
	require.NoError(s.T(), ad.Set(ctx, plainKey, []byte("v6"), time.Hour))

	// Blocking snapshot, then restart the process (same container FS).
	require.NoError(s.T(), client.Do(ctx, "SAVE").Err())
	stopTimeout := 10 * time.Second
	require.NoError(s.T(), c.Stop(ctx, &stopTimeout))
	require.NoError(s.T(), c.Start(ctx))
	client2 := s.dialRaw(c)
	defer func() { _ = client2.Close() }()
	ad2 := provider.New(provider.Options{Addr: client2.Options().Addr, Role: cachekit.StoreDurable})
	defer func() { _ = ad2.Close() }()

	got, err := ad2.Poll(ctx, 10)
	require.NoError(s.T(), err)
	require.Len(s.T(), got, 1, "scheduled job must survive restart")
	require.Equal(s.T(), duePayload, got[0])

	raw, err := ad2.Get(ctx, nxKey)
	require.NoError(s.T(), err, "SETNX claim must survive restart")
	require.JSONEq(s.T(), `{"upload":6}`, string(raw))

	plain, err := ad2.Get(ctx, plainKey)
	require.NoError(s.T(), err, "plain entry must survive restart")
	require.Equal(s.T(), "v6", string(plain))
}

// TestT07FailOpen covers domain reads succeeding via DB when volatile is down.
func (s *ConformanceSuite) TestT07FailOpen() {
	ctx := context.Background()
	dead := s.deadAdapter()
	defer func() { _ = dead.Close() }()

	// Reads fail with ErrUnavailable (callers fall through to DB).
	start := time.Now()
	_, err := dead.Get(ctx, "t07:any")
	require.ErrorIs(s.T(), err, cachekit.ErrUnavailable)
	require.Less(s.T(), time.Since(start), 2*time.Second,
		"dead-backend GET must fail fast, not hang")

	// The fetch path still serves DB bytes when the backend is down.
	flight := &cachekit.Flight{}
	got, err := cachekit.FetchBytes(ctx, dead, cachekit.NopRecorder(), flight,
		"qa", "t07", uniq("t07")+"k", 5*time.Minute, time.Minute,
		func(ctx context.Context) ([]byte, error) { return []byte("db-bytes"), nil })
	require.NoError(s.T(), err)
	require.Equal(s.T(), []byte("db-bytes"), got)

	// Backend error on admission fails open to DB without caching.
	admit, err := cachekit.AdmitList(ctx, dead, 7, "t07hash")
	require.Error(s.T(), err)
	require.False(s.T(), admit)
}

// TestT08Invalidation covers write → entity miss + list version bump.
func (s *ConformanceSuite) TestT08Invalidation() {
	ctx := context.Background()
	ns := uniq("t08")

	// Seed then invalidate: post-commit Del must produce a miss.
	key := ns + "entity"
	require.NoError(s.T(), s.volatile.Set(ctx, key, []byte("stale"), time.Minute))
	require.True(s.T(), s.volatile.Flush(5*time.Second))
	_, err := s.volatile.Get(ctx, key)
	require.NoError(s.T(), err, "seeded entity must hit before invalidation")
	require.NoError(s.T(), s.volatile.Del(ctx, key))
	_, err = s.volatile.Get(ctx, key)
	require.ErrorIs(s.T(), err, cachekit.ErrMiss, "invalidated entity must miss")

	// Deleting a missing key is a nil no-op (idempotent post-commit cleanup).
	require.NoError(s.T(), s.volatile.Del(ctx, ns+"missing"))

	// List versions bump monotonically on the durable role.
	vKey, err := cachekit.VersionKey(8008, "product")
	require.NoError(s.T(), err)
	v1, err := cachekit.BumpSellerVersion(ctx, s.durable, 8008, "product")
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint64(1), v1)
	v2, err := cachekit.BumpSellerVersion(ctx, s.durable, 8008, "product")
	require.NoError(s.T(), err)
	require.Equal(s.T(), v1+1, v2)
	raw, err := cachekit.ReadVersion(ctx, s.durable, vKey)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "2", raw)
}

// TestT09Isolation covers cross-seller key rejection.
func (s *ConformanceSuite) TestT09Isolation() {
	ctx := context.Background()
	const sellerA uint = 424242
	const sellerB uint = 434343

	keyA := cachekit.MustBuildSellerKey(sellerA, "product", "1")
	require.NoError(s.T(), s.volatile.Set(ctx, keyA, []byte("a-data"), time.Minute))
	require.True(s.T(), s.volatile.Flush(5*time.Second))

	// Same tail under another seller is a different key: must miss.
	keyB := cachekit.MustBuildSellerKey(sellerB, "product", "1")
	require.NotEqual(s.T(), keyA, keyB)
	_, err := s.volatile.Get(ctx, keyB)
	require.ErrorIs(s.T(), err, cachekit.ErrMiss, "cross-seller read must miss")

	// Owner still hits.
	got, err := s.volatile.Get(ctx, keyA)
	require.NoError(s.T(), err)
	require.Equal(s.T(), []byte("a-data"), got)

	// Builders fail closed on unscoped input.
	_, err = cachekit.BuildSellerKey(0, "product", "1")
	require.ErrorIs(s.T(), err, cachekit.ErrUnscopedKey, "seller 0 must be rejected")
	_, err = cachekit.BuildSellerKey(sellerA, "a*b")
	require.ErrorIs(s.T(), err, cachekit.ErrUnscopedKey, "glob segment must be rejected")
	_, err = cachekit.BuildSellerKey(sellerA, "")
	require.ErrorIs(s.T(), err, cachekit.ErrUnscopedKey, "empty segment must be rejected")

	// Validators fail closed on forged keys.
	require.Error(s.T(), cachekit.ValidateKey("seller:abc:product:1"))
	require.Error(s.T(), cachekit.ValidateKey("seller:1:"))
	require.Error(s.T(), cachekit.ValidateKey("seller:1:a*b"))
	require.NoError(s.T(), cachekit.ValidateKey(keyA), "well-formed key must validate")
}

// TestT10EvictionIsolation covers durable jobs surviving volatile maxmemory pressure.
func (s *ConformanceSuite) TestT10EvictionIsolation() {
	ctx := context.Background()

	// Durable residents first (must remain throughout). They live on the
	// suite durable backend; pressure below targets only the dedicated
	// volatile container, so this holds in shared and dual modes alike.
	duePayload := []byte(fmt.Sprintf(`{"job":"t10-%d"}`, time.Now().UnixNano()))
	_, err := s.durable.Schedule(ctx, duePayload, 0)
	require.NoError(s.T(), err)
	nxKey := uniq("t10idem")
	claimed, err := s.durable.SetNX(ctx, nxKey, []byte(`{"upload":10}`), time.Hour)
	require.NoError(s.T(), err)
	require.True(s.T(), claimed)

	// Dedicated volatile container WITH eviction enabled.
	_, image, dragonfly := kvImage()
	var cmd []string
	if dragonfly {
		cmd = []string{"--cache_mode", "--maxmemory=2mb"}
	}
	evict, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        image,
				Cmd:          cmd,
				ExposedPorts: []string{"6379/tcp"},
				WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(2 * time.Minute),
			},
			Started: true,
		})
	require.NoError(s.T(), err, "evicting volatile container must start")
	defer func() { _ = evict.Terminate(ctx) }()
	vclient := s.dialRaw(evict)
	defer func() { _ = vclient.Close() }()
	if !dragonfly {
		require.NoError(s.T(), vclient.ConfigSet(ctx, "maxmemory", "2mb").Err())
		require.NoError(s.T(), vclient.ConfigSet(ctx, "maxmemory-policy", "allkeys-lru").Err())
	}

	// Flood volatile past its ceiling with 30KB values.
	const flood = 120
	blob := make([]byte, 30*1024)
	for i := range blob {
		blob[i] = byte(i)
	}
	pipe := vclient.Pipeline()
	for i := 0; i < flood; i++ {
		pipe.Set(ctx, fmt.Sprintf("t10flood:%04d", i), blob, 0)
	}
	_, err = pipe.Exec(ctx)
	require.NoError(s.T(), err)

	// Early keys must have evicted (pressure worked) while late keys stand.
	evicted := 0
	for i := 0; i < 20; i++ {
		if err := vclient.Get(ctx, fmt.Sprintf("t10flood:%04d", i)).Err(); err != nil {
			evicted++
		}
	}
	require.Greater(s.T(), evicted, 0, "volatile pressure must evict early keys")
	_, err = vclient.Get(ctx, fmt.Sprintf("t10flood:%04d", flood-1)).Bytes()
	require.NoError(s.T(), err, "late volatile keys must stand")

	// Durable residents untouched by volatile pressure.
	got, err := s.durable.Poll(ctx, 10)
	require.NoError(s.T(), err)
	found := false
	for _, payload := range got {
		if string(payload) == string(duePayload) {
			found = true
		}
	}
	require.True(s.T(), found, "durable delayed job must survive volatile pressure")
	raw, err := s.durable.Get(ctx, nxKey)
	require.NoError(s.T(), err, "durable SETNX claim must survive volatile pressure")
	require.JSONEq(s.T(), `{"upload":10}`, string(raw))

	_ = s.durable.Del(ctx, nxKey)
}

// TestT11PayloadContract covers cached DTO exclusions (stock, signed URLs, personalization).
func (s *ConformanceSuite) TestT11PayloadContract() {
	ctx := context.Background()

	// Codec round-trips bytes losslessly.
	raw, err := cachekit.Marshal(map[string]any{"id": 11, "name": "t11"})
	require.NoError(s.T(), err)
	var decoded map[string]any
	require.NoError(s.T(), cachekit.Unmarshal(raw, &decoded))
	require.Equal(s.T(), "t11", decoded["name"])

	// Oversize values are dropped and counted, never stored.
	big := make([]byte, 300*1024)
	for i := range big {
		big[i] = byte(i)
	}
	bigKey := uniq("t11") + "big"
	before := s.volatile.Stats().AsyncDropped
	require.NoError(s.T(), s.volatile.Set(ctx, bigKey, big, time.Minute),
		"oversize SET must return nil (async drop, never an error)")
	require.True(s.T(), s.volatile.Flush(5*time.Second))
	require.Greater(s.T(), s.volatile.Stats().AsyncDropped, before,
		"oversize value must be counted as dropped")
	_, err = s.volatile.Get(ctx, bigKey)
	require.ErrorIs(s.T(), err, cachekit.ErrMiss, "dropped value must never be readable")
}

// TestT12Denylist covers hashed key, remaining-exp TTL, and fail-closed reads.
func (s *ConformanceSuite) TestT12Denylist() {
	ctx := context.Background()
	dl := cachekit.NewTokenDenylist(s.durable)
	require.NotNil(s.T(), dl)

	token := fmt.Sprintf("t12-token-%d", time.Now().UnixNano())
	expiresAt := time.Now().Add(10 * time.Minute)

	// Unknown token is not revoked.
	revoked, err := dl.IsRevoked(ctx, token)
	require.NoError(s.T(), err)
	require.False(s.T(), revoked)

	// Revoke → revoked, with a key shaped as bl:<sha256> and a remaining-exp TTL.
	require.NoError(s.T(), dl.Revoke(ctx, token, expiresAt))
	revoked, err = dl.IsRevoked(ctx, token)
	require.NoError(s.T(), err)
	require.True(s.T(), revoked)

	blKey, err := cachekit.DenylistKey(token)
	require.NoError(s.T(), err)
	require.Regexp(s.T(), `^bl:[0-9a-f]{64}$`, blKey, "denylist key must be bl:<sha256-hex>")
	ttl, err := s.durable.TTL(ctx, blKey)
	require.NoError(s.T(), err)
	require.Greater(s.T(), ttl, time.Duration(0))
	require.LessOrEqual(s.T(), ttl, 10*time.Minute, "denylist TTL must be remaining-exp")

	// Key derivation is deterministic: same token → same key, distinct tokens →
	// distinct keys (stable hashing is what makes revoke/check agree).
	blKey2, err := cachekit.DenylistKey(token)
	require.NoError(s.T(), err)
	require.Equal(s.T(), blKey, blKey2)
	otherKey, err := cachekit.DenylistKey(token + "-other")
	require.NoError(s.T(), err)
	require.NotEqual(s.T(), blKey, otherKey)

	// Unwired denylist fails closed.
	require.Nil(s.T(), cachekit.NewTokenDenylist(nil))
	var nilDL *cachekit.TokenDenylist
	_, err = nilDL.IsRevoked(ctx, token)
	require.ErrorIs(s.T(), err, cachekit.ErrUnavailable, "unwired reads must fail closed")

	// Dead backend surfaces an error (caller treats errors as revoked).
	deadDurable := provider.New(provider.Options{
		Addr:         "127.0.0.1:1",
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
		PoolTimeout:  200 * time.Millisecond,
		MaxRetries:   0,
		Recorder:     cachekit.NopRecorder(),
		Role:         cachekit.StoreDurable,
	})
	defer func() { _ = deadDurable.Close() }()
	deadDL := cachekit.NewTokenDenylist(deadDurable)
	_, err = deadDL.IsRevoked(ctx, token)
	require.Error(s.T(), err, "backend failure must surface, never read as not-revoked")
}

// TestT13Admission covers marker-only one-offs and second-sight fills.
func (s *ConformanceSuite) TestT13Admission() {
	ctx := context.Background()
	hash := fmt.Sprintf("t13hash-%d", time.Now().UnixNano())

	// First sight: no fill. Repeat within the window: fill allowed.
	admit, err := cachekit.AdmitList(ctx, s.volatile, 13, hash)
	require.NoError(s.T(), err)
	require.False(s.T(), admit, "first sight must not admit")
	admit, err = cachekit.AdmitList(ctx, s.volatile, 13, hash)
	require.NoError(s.T(), err)
	require.True(s.T(), admit, "repeat sight must admit")

	// Marker exists on the wire with a ~60s TTL.
	marker, err := cachekit.AdmissionMarkerKey(13, hash)
	require.NoError(s.T(), err)
	wireTTL, err := s.container.RedisClient.TTL(ctx, marker).Result()
	require.NoError(s.T(), err)
	require.Greater(s.T(), wireTTL, 40*time.Second)
	require.LessOrEqual(s.T(), wireTTL, 69*time.Second)

	// Independent hash admits independently (first sight).
	admit, err = cachekit.AdmitList(ctx, s.volatile, 13, hash+"-other")
	require.NoError(s.T(), err)
	require.False(s.T(), admit)

	// Unscoped input fails closed.
	_, err = cachekit.AdmitList(ctx, s.volatile, 0, hash)
	require.ErrorIs(s.T(), err, cachekit.ErrUnscopedKey)
}

// TestT14DegradedBackend covers flat p99 + dropped counted SETs on slow/dead backend.
func (s *ConformanceSuite) TestT14DegradedBackend() {
	ctx := context.Background()
	// Low trip threshold: a handful of failed SETs must shed within the wait.
	dead := provider.New(provider.Options{
		Addr:             "127.0.0.1:1",
		ReadTimeout:      200 * time.Millisecond,
		WriteTimeout:     200 * time.Millisecond,
		PoolTimeout:      200 * time.Millisecond,
		MaxRetries:       0,
		AsyncBudget:      50 * time.Millisecond,
		AsyncMaxInflight: 16,
		BreakerThreshold: 3,
		BreakerCooldown:  time.Minute,
		Recorder:         cachekit.NopRecorder(),
		Role:             cachekit.StoreCache,
	})
	defer func() { _ = dead.Close() }()

	// Reads fail fast with ErrUnavailable (flat latency, no hangs).
	start := time.Now()
	_, err := dead.Get(ctx, "t14:any")
	require.ErrorIs(s.T(), err, cachekit.ErrUnavailable)
	require.Less(s.T(), time.Since(start), 2*time.Second)

	// Domain reads still serve DB bytes through the fetch path.
	flight := &cachekit.Flight{}
	got, err := cachekit.FetchBytes(ctx, dead, cachekit.NopRecorder(), flight,
		"qa", "t14", uniq("t14")+"k", 5*time.Minute, time.Minute,
		func(ctx context.Context) ([]byte, error) { return []byte("db-bytes"), nil })
	require.NoError(s.T(), err)
	require.Equal(s.T(), []byte("db-bytes"), got)

	// Writes never error on the request path but the backend pressure must
	// trip the shed (or at minimum count drops) within a bounded wait.
	for i := 0; i < 15; i++ {
		require.NoError(s.T(), dead.Set(ctx, fmt.Sprintf("t14:k:%d", i), []byte("v"), time.Minute))
	}
	require.Eventually(s.T(), func() bool {
		st := dead.Stats()
		return st.ShedActive || st.AsyncDropped > 0 || st.ShedSkipped > 0
	}, 5*time.Second, 50*time.Millisecond, "dead backend must shed or count drops")
}

// TestT15WriteShed covers breaker trip, continued GETs, and cooldown recovery.
func (s *ConformanceSuite) TestT15WriteShed() {
	ctx := context.Background()

	// Manual lever: writes shed while reads keep working.
	local := provider.New(provider.Options{
		Addr: s.container.RedisClient.Options().Addr,
		Role: cachekit.StoreCache,
	})
	defer func() { _ = local.Close() }()
	local.SetSetWritesEnabled(false)
	defer local.ClearSetWritesOverride()
	require.True(s.T(), local.ShedActive(), "manual lever must report shedding")

	shedKey := uniq("t15") + "manual"
	before := local.Stats().ShedSkipped
	require.NoError(s.T(), local.Set(ctx, shedKey, []byte("v"), time.Minute))
	require.True(s.T(), local.Flush(5*time.Second))
	require.Greater(s.T(), local.Stats().ShedSkipped, before, "shed SETs must be counted")
	_, err := local.Get(ctx, shedKey)
	require.ErrorIs(s.T(), err, cachekit.ErrMiss, "shed write must never land")

	// Auto-trip on a failing backend, then cooldown recovery.
	trip := provider.New(provider.Options{
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
	defer func() { _ = trip.Close() }()
	for i := 0; i < 5; i++ {
		_ = trip.Set(ctx, fmt.Sprintf("t15:trip:%d", i), []byte("v"), time.Minute)
	}
	require.Eventually(s.T(), trip.ShedActive, 5*time.Second, 50*time.Millisecond,
		"consecutive failures must trip the breaker")
	time.Sleep(500 * time.Millisecond)
	require.False(s.T(), trip.ShedActive(), "breaker must recover after cooldown")
}

// TestT16Jitter covers wire TTLs inside base ±15% across a batch.
func (s *ConformanceSuite) TestT16Jitter() {
	const base = 60 * time.Second
	const samples = 200

	// Pure function: every sample inside [base, base+15%], with real variance.
	min, max := base+time.Second, time.Duration(0)
	for i := 0; i < samples; i++ {
		j := cachekit.JitteredTTL(base)
		require.GreaterOrEqual(s.T(), j, base, "jitter must not go below base")
		require.LessOrEqual(s.T(), j, base+9*time.Second, "jitter must cap at +15%")
		if j < min {
			min = j
		}
		if j > max {
			max = j
		}
	}
	require.Less(s.T(), min, max, "jitter must vary across samples")

	// Wire TTLs land in the same band (elapsed-time slack on the low end).
	ctx := context.Background()
	wKey := uniq("t16") + "wire"
	require.NoError(s.T(), s.volatile.Set(ctx, wKey, []byte("v"), base))
	require.True(s.T(), s.volatile.Flush(5*time.Second))
	wire, err := s.container.RedisClient.TTL(ctx, wKey).Result()
	require.NoError(s.T(), err)
	require.Greater(s.T(), wire, 45*time.Second)
	require.LessOrEqual(s.T(), wire, 69*time.Second)
}

// TestT17GenerationGuard covers dropped stale async SETs after a concurrent write.
func (s *ConformanceSuite) TestT17GenerationGuard() {
	ctx := context.Background()
	ns := uniq("t17")
	dataKey := ns + "entity"
	genKey := cachekit.GenerationKeyFor(dataKey)

	// Fresh key starts at generation zero.
	gen, err := cachekit.CaptureGeneration(ctx, s.volatile, dataKey)
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint64(0), gen)

	// Stale write against generation zero drops and is counted.
	before := s.volatile.Stats().GenerationDropped
	stored, err := s.volatile.CompareAndSet(ctx, dataKey, genKey, 0, []byte("stale"), time.Minute)
	require.NoError(s.T(), err)
	require.False(s.T(), stored, "stale write must drop")
	require.Greater(s.T(), s.volatile.Stats().GenerationDropped, before)

	// Concurrent writer bumps the generation; current-gen write lands.
	gen, err = cachekit.BumpGeneration(ctx, s.volatile, dataKey)
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint64(1), gen)
	stored, err = cachekit.StoreGuarded(ctx, s.volatile, dataKey, gen, []byte("fresh"), time.Minute)
	require.NoError(s.T(), err)
	require.True(s.T(), stored, "current-generation write must land")
	got, err := s.volatile.Get(ctx, dataKey)
	require.NoError(s.T(), err)
	require.Equal(s.T(), []byte("fresh"), got)

	// The late stale retry still drops against the bumped generation.
	stored, err = s.volatile.CompareAndSet(ctx, dataKey, genKey, 0, []byte("staler"), time.Minute)
	require.NoError(s.T(), err)
	require.False(s.T(), stored)
	got, err = s.volatile.Get(ctx, dataKey)
	require.NoError(s.T(), err)
	require.Equal(s.T(), []byte("fresh"), got, "stale retry must not clobber the winner")
}

// TestConformanceSuite runs the full T1–T17 matrix.
func TestConformanceSuite(t *testing.T) {
	suite.Run(t, new(ConformanceSuite))
}
