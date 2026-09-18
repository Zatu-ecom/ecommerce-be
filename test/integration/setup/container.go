package setup

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
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
	ctx             context.Context
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

// kvWaitStrategy returns a readiness wait per backend. Redis reports a ready
// log line; Dragonfly readiness is port-based (log text varies by version),
// followed by a PING loop in pingKV below for both.
func kvWaitStrategy(backend string) wait.Strategy {
	if backend == "dragonfly" {
		return wait.ForListeningPort("6379/tcp").
			WithStartupTimeout(2 * time.Minute)
	}
	return wait.ForLog("Ready to accept connections").
		WithStartupTimeout(2 * time.Minute)
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

// kvClient dials a KV container and waits until PING succeeds (covers the gap
// between port-open/log-ready and actual command readiness).
func kvClient(t *testing.T, ctx context.Context, c testcontainers.Container, role string) *redis.Client {
	t.Helper()
	host, err := c.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get %s host: %v", role, err)
	}
	port, err := c.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get %s port: %v", role, err)
	}
	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port.Port()),
	})
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if err := client.Ping(ctx).Err(); err == nil {
			return client
		} else if time.Now().After(deadline) {
			t.Fatalf("%s not responding to PING: %v", role, err)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// SetupTestContainers sets up the test containers for Postgres and Redis
func SetupTestContainers(t *testing.T) *TestContainer {
	ctx := context.Background()

	// Postgres container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(5*time.Minute),
				wait.ForListeningPort("5432/tcp").
					WithStartupTimeout(5*time.Minute),
			),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// KV containers (volatile + durable roles, selected backend image)
	backendName, backendImage := kvBackend()
	redisContainer := startKVContainer(t, ctx, backendName, backendImage)
	durableContainer := startKVContainer(t, ctx, backendName, backendImage)

	// Get Postgres connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable&TimeZone=UTC")
	if err != nil {
		t.Fatalf("failed to get postgres connection string: %v", err)
	}

	// Connect to the database with GORM (retry: postgres can report ready before accepting TCP)
	var gormDB *gorm.DB
	const maxDBAttempts = 15
	// Brief pause after the log-based wait — under parallel test load the port
	// can still refuse connections for a moment.
	time.Sleep(500 * time.Millisecond)
	for attempt := 1; attempt <= maxDBAttempts; attempt++ {
		gormDB, err = gorm.Open(gormpostgres.Open(connStr), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true, // Use singular table names to match production
			},
		})
		if err == nil {
			sqlDB, dbErr := gormDB.DB()
			if dbErr == nil {
				pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				dbErr = sqlDB.PingContext(pingCtx)
				cancel()
			}
			if dbErr == nil {
				break
			}
			err = dbErr
		}
		if attempt == maxDBAttempts {
			t.Fatalf("failed to connect to database after %d attempts: %v", maxDBAttempts, err)
		}
		time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
	}

	// Get Redis connection details
	redisClient := kvClient(t, ctx, redisContainer, "redis")
	durableClient := kvClient(t, ctx, durableContainer, "durable-kv")

	return &TestContainer{
		Postgres:        pgContainer,
		Redis:           redisContainer,
		DurableKV:       durableContainer,
		DB:              gormDB,
		RedisClient:     redisClient,
		DurableKVClient: durableClient,
		KVBackendName:   backendName,
		ctx:             ctx,
	}
}

// Cleanup terminates the test containers
func (tc *TestContainer) Cleanup(t *testing.T) {
	if err := tc.Postgres.Terminate(tc.ctx); err != nil {
		t.Logf("failed to terminate postgres container: %v", err)
	}
	if err := tc.Redis.Terminate(tc.ctx); err != nil {
		t.Logf("failed to terminate redis container: %v", err)
	}
	if tc.DurableKV != nil {
		if err := tc.DurableKV.Terminate(tc.ctx); err != nil {
			t.Logf("failed to terminate durable-kv container: %v", err)
		}
	}

	// Reset singleton factories so later packages do not reuse stale services.
	ResetAllModuleSingletons()
}
