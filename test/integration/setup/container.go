package setup

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
)

// sharedKV is one Redis per test process. Docker Desktop's published-port
// NAT breaks after dozens of create/destroy cycles (host maps a port, the
// VM never listens, PING refuses for minutes). Reusing one container for
// the package avoids that; Ryuk still reaps it when the process exits.
var (
	sharedKVMu     sync.Mutex
	sharedKV       testcontainers.Container
	sharedKVClient *redis.Client
)

// TestContainer holds the containers for testing
type TestContainer struct {
	Postgres *postgres.PostgresContainer
	Redis    testcontainers.Container
	// DurableKV is the second KV container (durable role: scheduler queue,
	// idempotency, denylist, limiter, version counters). Same backend image
	// as Redis unless KV_BACKEND selects otherwise per-role in the future.
	DurableKV   testcontainers.Container
	DB          *gorm.DB
	RedisClient *redis.Client
	// DurableKVClient dials DurableKV (used by durable-path conformance tests).
	DurableKVClient *redis.Client
	KVBackendName   string
	// DualKV is true when TEST_KV_DUAL=1 started a second Redis process.
	// Default is false: CACHE_ADDR and KV_ADDR share one container.
	DualKV bool
	ctx    context.Context
}

// dualKVRequested is true when tests opt into two Redis processes (volatile
// vs durable isolation). Unset is the local default: one Redis for both addrs.
func dualKVRequested() bool {
	v := os.Getenv("TEST_KV_DUAL")
	return v == "1" || v == "true" || v == "TRUE"
}

// SharedKV reports that volatile and durable roles share one container.
func (tc *TestContainer) SharedKV() bool {
	return tc == nil || !tc.DualKV
}

// RequireDualKV skips the calling test unless TEST_KV_DUAL=1.
func (tc *TestContainer) RequireDualKV(t *testing.T) {
	t.Helper()
	if tc.SharedKV() {
		t.Skip("requires TEST_KV_DUAL=1 (separate volatile and durable Redis)")
	}
}

// kvBackend resolves which KV image to run. Supported: "redis" (default),
// "dragonfly". Selected via KV_BACKEND env so the T1-T17 conformance suite
// runs unmodified against both backends (012 cutover gate).
func kvBackend() (name, image string) {
	switch os.Getenv("KV_BACKEND") {
	case "dragonfly":
		return "dragonfly", "docker.dragonflydb.io/dragonflydb/dragonfly:v1.40.0"
	default:
		return "redis", "redis:7-alpine"
	}
}

// kvWaitStrategy waits until the *host-mapped* port accepts TCP. A log-only
// wait is not enough on Docker Desktop: Redis prints "Ready" inside the
// container while 127.0.0.1:<published> still refuses.
func kvWaitStrategy(backend string) wait.Strategy {
	port := wait.ForListeningPort("6379/tcp").WithStartupTimeout(2 * time.Minute)
	if backend == "dragonfly" {
		return port
	}
	return wait.ForAll(
		wait.ForLog("Ready to accept connections").WithStartupTimeout(2*time.Minute),
		port,
	)
}

// startKVContainer starts one KV role container for the selected backend.
func startKVContainer(t *testing.T, ctx context.Context, backend, image string) testcontainers.Container {
	t.Helper()
	c, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        image,
				ExposedPorts: []string{"6379/tcp"},
				WaitingFor:   kvWaitStrategy(backend),
			},
			Started: true,
		},
	)
	if err != nil {
		t.Fatalf("failed to start %s container: %v", backend, err)
	}
	return c
}

// kvClient dials a KV container and waits until PING succeeds. Deadline is
// short: if Docker Desktop never published the port, waiting 5 minutes only
// floods logs (go-redis retries) and does not recover.
func kvClient(t *testing.T, ctx context.Context, c testcontainers.Container, role string) *redis.Client {
	t.Helper()
	host, err := c.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get %s host: %v", role, err)
	}
	if host == "localhost" {
		host = "127.0.0.1"
	}
	port, err := c.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get %s port: %v", role, err)
	}
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port.Port()),
		MaxRetries:   0,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	deadline := time.Now().Add(20 * time.Second)
	var last error
	for {
		pingCtx, cancel := context.WithTimeout(ctx, time.Second)
		last = client.Ping(pingCtx).Err()
		cancel()
		if last == nil {
			return client
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s not responding to PING on %s: %v", role, client.Options().Addr, last)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// acquireSharedKV returns the process-wide Redis, starting it on first use.
func acquireSharedKV(t *testing.T, ctx context.Context, backend, image string) (testcontainers.Container, *redis.Client) {
	t.Helper()
	sharedKVMu.Lock()
	defer sharedKVMu.Unlock()

	if sharedKVClient != nil {
		pingCtx, cancel := context.WithTimeout(ctx, time.Second)
		err := sharedKVClient.Ping(pingCtx).Err()
		cancel()
		if err == nil {
			_ = sharedKVClient.FlushAll(ctx).Err()
			return sharedKV, sharedKVClient
		}
		t.Logf("shared redis died (%v); starting a replacement", err)
		_ = sharedKV.Terminate(ctx)
		sharedKV, sharedKVClient = nil, nil
	}

	c := startKVContainer(t, ctx, backend, image)
	client := kvClient(t, ctx, c, "redis")
	sharedKV = c
	sharedKVClient = client
	return c, client
}

// SetupTestContainers returns a handle on the process-wide Postgres and Redis
// backends (one container per `go test` package process, reused across suites).
// On reuse the Postgres tables are truncated and Redis is flushed, which is
// the data-equivalent of the previous fresh-container-per-suite behavior.
// Callers keep their existing RunAllMigrations/RunAllSeeds calls: migrations
// run once per process (guarded), seeds are upsert-safe and repopulate.
// Set TEST_USE_EXTERNAL=1 to dial externally-provisioned backends
// (docker-compose.test.yml) instead of starting Testcontainers.
func SetupTestContainers(t *testing.T) *TestContainer {
	ctx := context.Background()

	// Postgres: process-wide shared handle (container or external).
	pgContainer, gormDB, _ := acquireSharedPG(t, ctx)

	// KV: one Redis by default (CACHE_ADDR and KV_ADDR share it), reused for
	// the whole test process so Docker Desktop is not asked to publish a new
	// host port per TestXxx. TEST_KV_DUAL=1 starts exclusive containers so
	// isolation tests can Stop one role. TEST_USE_EXTERNAL=1 dials
	// externally-provisioned Redis instead of starting containers.
	backendName, backendImage := kvBackend()
	dual := dualKVRequested()
	external := useExternalTestDB()
	var redisContainer, durableContainer testcontainers.Container
	var redisClient, durableClient *redis.Client
	switch {
	case external && !dual:
		redisClient = dialExternalRedis(t, ctx, externalRedisAddr("TEST_CACHE", "6382"), "cache")
		durableClient = dialExternalRedis(t, ctx, externalRedisAddr("TEST_DURABLE", "6383"), "durable-kv")
		_ = redisClient.FlushAll(ctx).Err()
		_ = durableClient.FlushAll(ctx).Err()
	case dual:
		redisContainer = startKVContainer(t, ctx, backendName, backendImage)
		durableContainer = startKVContainer(t, ctx, backendName, backendImage)
		redisClient = kvClient(t, ctx, redisContainer, "redis")
		durableClient = kvClient(t, ctx, durableContainer, "durable-kv")
	default:
		redisContainer, redisClient = acquireSharedKV(t, ctx, backendName, backendImage)
		durableContainer = redisContainer
		durableClient = redis.NewClient(&redis.Options{
			Addr:         redisClient.Options().Addr,
			MaxRetries:   0,
			DialTimeout:  500 * time.Millisecond,
			ReadTimeout:  2 * time.Second,
			WriteTimeout: 2 * time.Second,
		})
	}

	// Publish role addrs for config-driven wiring: module factories build
	// their durable queues from KV_ADDR (scheduler.WiringQueue), and future
	// strategies will resolve CACHE_ADDR the same way. Per-suite overwrite is
	// safe: singletons reset and config reloads in SetupTestServer.
	os.Setenv("CACHE_ADDR", redisClient.Options().Addr)
	os.Setenv("KV_ADDR", durableClient.Options().Addr)

	return &TestContainer{
		Postgres:        pgContainer,
		Redis:           redisContainer,
		DurableKV:       durableContainer,
		DB:              gormDB,
		RedisClient:     redisClient,
		DurableKVClient: durableClient,
		KVBackendName:   backendName,
		DualKV:          dual,
		ctx:             ctx,
	}
}

// Cleanup releases per-handle resources. Process-wide shared Postgres/Redis
// (and external backends) stay up for the next suite in this process; Ryuk
// removes containers when the package process exits. Only exclusive dual-KV
// containers are terminated here.
func (tc *TestContainer) Cleanup(t *testing.T) {
	if tc == nil {
		return
	}
	// Exclusive (dual) Redis is torn down with the test. The process-wide
	// shared Redis stays up so the next TestXxx does not need a new published
	// port; Ryuk removes it when this package process exits.
	if tc.DualKV {
		if tc.Redis != nil {
			if err := tc.Redis.Terminate(tc.ctx); err != nil {
				t.Logf("failed to terminate redis container: %v", err)
			}
		}
		if tc.DurableKV != nil && tc.DurableKV != tc.Redis {
			if err := tc.DurableKV.Terminate(tc.ctx); err != nil {
				t.Logf("failed to terminate durable-kv container: %v", err)
			}
		}
	}

	// Reset singleton factories so later packages do not reuse stale services.
	ResetAllModuleSingletons()
}
