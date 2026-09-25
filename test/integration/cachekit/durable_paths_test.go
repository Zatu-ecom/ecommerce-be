// Package cachekit_test — durable-path conformance fill-in (T057, H3).
//
// Full T2/T4/T5/T6/T10 implementations against the backend selected by
// KV_BACKEND (T017 provided skeletons). Durable KV is the non-evicting,
// snapshotted role: idempotency, scheduler transport, version counters must
// survive eviction pressure elsewhere and restarts.
package cachekit_test

import (
	"context"
	"fmt"
	"os"
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

// DurablePathsSuite exercises T2/T4/T5/T6/T10 on real backends.
type DurablePathsSuite struct {
	suite.Suite
	container *setup.TestContainer
	durable   cachekit.DurableQueue
	volatile  *provider.Adapter
}

func (s *DurablePathsSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	durableAddr := s.container.DurableKVClient.Options().Addr
	s.durable = provider.New(provider.Options{Addr: durableAddr, Role: cachekit.StoreDurable})
	volatileAddr := s.container.RedisClient.Options().Addr
	s.volatile = provider.New(provider.Options{Addr: volatileAddr, Role: cachekit.StoreCache})
}

func (s *DurablePathsSuite) TearDownSuite() {
	s.container.Cleanup(s.T())
}

// kvImage mirrors the backend selection in setup (same pinned tags).
func kvImage() (backend, image string, dragonfly bool) {
	if os.Getenv("KV_BACKEND") == "dragonfly" {
		return "dragonfly", "docker.dragonflydb.io/dragonflydb/dragonfly:v1.40.0", true
	}
	return "redis", "redis:7-alpine", false
}

// TestT02SetNX_IdempotentClaim proves exactly-once claims with replay:
// winner true, losers false, release re-arms — on two clients.
func (s *DurablePathsSuite) TestT02SetNX_IdempotentClaim() {
	ctx := context.Background()
	key := fmt.Sprintf("file:init:idem:test:t02-%d", time.Now().UnixNano())
	_ = s.durable.Del(ctx, key)

	claimed, err := s.durable.SetNX(ctx, key, []byte(`{"upload":1}`), time.Minute)
	require.NoError(s.T(), err)
	require.True(s.T(), claimed, "first claim must win")

	replay, err := s.durable.SetNX(ctx, key, []byte(`{"upload":2}`), time.Minute)
	require.NoError(s.T(), err)
	require.False(s.T(), replay, "replay must lose")

	got, err := s.durable.Get(ctx, key)
	require.NoError(s.T(), err)
	require.JSONEq(s.T(), `{"upload":1}`, string(got), "winner payload must stand")

	// Second client on the same backend agrees.
	client2 := provider.New(provider.Options{Addr: s.container.DurableKVClient.Options().Addr, Role: cachekit.StoreDurable})
	defer client2.Close()
	replay2, err := client2.SetNX(ctx, key, []byte(`{"upload":3}`), time.Minute)
	require.NoError(s.T(), err)
	require.False(s.T(), replay2, "second client must lose while claimed")

	require.NoError(s.T(), s.durable.Del(ctx, key))
	rearmed, err := client2.SetNX(ctx, key, []byte(`{"upload":4}`), time.Minute)
	require.NoError(s.T(), err)
	require.True(s.T(), rearmed, "delete must re-arm the claim")
	_ = s.durable.Del(ctx, key)
}

// TestT04DelayQueue_SchedulePollClaimCancel proves schedule → poll → atomic
// claim (exactly one claimant across clients) → cancel.
func (s *DurablePathsSuite) TestT04DelayQueue_SchedulePollClaimCancel() {
	ctx := context.Background()

	jobID, err := s.durable.Schedule(ctx, []byte(`{"job":"t04"}`), 0)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), jobID)

	got, err := s.durable.Poll(ctx, 10)
	require.NoError(s.T(), err)
	require.Len(s.T(), got, 1, "due job must poll exactly once")
	require.JSONEq(s.T(), `{"job":"t04"}`, string(got[0]))

	again, err := s.durable.Poll(ctx, 10)
	require.NoError(s.T(), err)
	require.Empty(s.T(), again, "claimed job must not re-poll (ZRem==1)")

	// Cancel removes a pending job: missing/already-run is a nil no-op.
	cancelID, err := s.durable.Schedule(ctx, []byte(`{"job":"t04-cancel"}`), time.Hour)
	require.NoError(s.T(), err)
	require.NoError(s.T(), s.durable.Cancel(ctx, cancelID))
	require.NoError(s.T(), s.durable.Cancel(ctx, "does-not-exist"))
	require.NoError(s.T(), s.durable.Cancel(ctx, jobID), "claimed job cancel is a no-op")
}

// TestT05ScanDelete_LargeKeyspace proves background SCAN prefix delete over
// ~2000 keys (the janitor path — never KEYS, never on the request path).
func (s *DurablePathsSuite) TestT05ScanDelete_LargeKeyspace() {
	ctx := context.Background()
	prefix := fmt.Sprintf("t05janitor:%d:", time.Now().UnixNano())
	const n = 2000

	pipe := s.container.RedisClient.Pipeline()
	for i := 0; i < n; i++ {
		pipe.Set(ctx, fmt.Sprintf("%s%04d", prefix, i), "v", 0)
	}
	_, err := pipe.Exec(ctx)
	require.NoError(s.T(), err)

	require.NoError(s.T(), s.volatile.DelPrefix(ctx, prefix))

	left, _, err := s.container.RedisClient.Scan(ctx, 0, prefix+"*", 1000).Result()
	require.NoError(s.T(), err)
	require.Empty(s.T(), left, "prefix delete must remove all %d keys", n)
}

// startPersistedContainer boots a dedicated backend container with
// persistence enabled (appendonly RDB for Redis; dir+dbfilename snapshots
// for Dragonfly) for the T6 restart test.
func (s *DurablePathsSuite) startPersistedContainer() testcontainers.Container {
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

// dialContainer builds a raw client for a container (re-resolved after
// restarts, which may remap ports) with PING readiness.
func (s *DurablePathsSuite) dialContainer(c testcontainers.Container) *redis.Client {
	s.T().Helper()
	ctx := context.Background()
	host, err := c.Host(ctx)
	require.NoError(s.T(), err)
	port, err := c.MappedPort(ctx, "6379")
	require.NoError(s.T(), err)
	client := redis.NewClient(&redis.Options{Addr: host + ":" + port.Port()})
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if err := client.Ping(ctx).Err(); err == nil {
			return client
		} else if time.Now().After(deadline) {
			s.T().Fatalf("persisted container not PINGing: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// TestT06Restart_Recovery proves delayed jobs + SETNX + plain keys survive
// a backend restart (durable role persistence): snapshot, stop, start,
// reclaim.
func (s *DurablePathsSuite) TestT06Restart_Recovery() {
	ctx := context.Background()
	c := s.startPersistedContainer()
	defer func() { _ = c.Terminate(ctx) }()

	client := s.dialContainer(c)
	ad := provider.New(provider.Options{Addr: client.Options().Addr, Role: cachekit.StoreDurable})
	defer ad.Close()

	// Due job + idempotency claim + plain entry, all on the durable role.
	duePayload := []byte(fmt.Sprintf(`{"job":"t06-%d"}`, time.Now().UnixNano()))
	jobID, err := ad.Schedule(ctx, duePayload, 0)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), jobID)
	nxKey := fmt.Sprintf("file:init:idem:test:t06-%d", time.Now().UnixNano())
	claimed, err := ad.SetNX(ctx, nxKey, []byte(`{"upload":6}`), time.Hour)
	require.NoError(s.T(), err)
	require.True(s.T(), claimed)
	plainKey := fmt.Sprintf("t06plain:%d", time.Now().UnixNano())
	require.NoError(s.T(), ad.Set(ctx, plainKey, []byte("v6"), time.Hour))

	// Blocking snapshot, then restart the process (same container FS).
	require.NoError(s.T(), client.Do(ctx, "SAVE").Err())
	stopTimeout := 10 * time.Second
	require.NoError(s.T(), c.Stop(ctx, &stopTimeout))
	require.NoError(s.T(), c.Start(ctx))
	client2 := s.dialContainer(c)
	defer client2.Close()
	ad2 := provider.New(provider.Options{Addr: client2.Options().Addr, Role: cachekit.StoreDurable})
	defer ad2.Close()

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

// TestT10EvictionIsolation proves the §0.3 split: volatile pressure evicts
// volatile keys while durable jobs/counters stay intact (today's LRU bug).
func (s *DurablePathsSuite) TestT10EvictionIsolation() {
	s.container.RequireDualKV(s.T())
	ctx := context.Background()

	// Durable residents first (must remain throughout).
	duePayload := []byte(fmt.Sprintf(`{"job":"t10-%d"}`, time.Now().UnixNano()))
	_, err := s.durable.Schedule(ctx, duePayload, 0)
	require.NoError(s.T(), err)
	nxKey := fmt.Sprintf("file:init:idem:test:t10-%d", time.Now().UnixNano())
	claimed, err := s.durable.SetNX(ctx, nxKey, []byte(`{"upload":10}`), time.Hour)
	require.NoError(s.T(), err)
	require.True(s.T(), claimed)

	// Dedicated volatile container WITH eviction enabled (testcontainers
	// defaults have none; Dragonfly needs --cache_mode at boot).
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
	vclient := s.dialContainer(evict)
	defer vclient.Close()
	if !dragonfly {
		// 2MB ceiling: the empty server already burns ~1MB overhead, so
		// 1MB would OOM every write (nothing to evict yet). 3.6MB of
		// payload against 2MB forces real LRU eviction instead.
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

func TestDurablePathsSuite(t *testing.T) {
	suite.Run(t, new(DurablePathsSuite))
}
