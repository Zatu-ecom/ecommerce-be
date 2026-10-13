package setup

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
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

// Shared Postgres lifecycle (test infra only, no production logic).
//
// Before: every SetupTestContainers() booted a fresh postgres:16-alpine
// container and every suite replayed all 31 migrations + seeds. With ~117
// call sites across ~20 packages run serially (-p 1), container startup +
// DDL dominated wall-clock time.
//
// After: one postgres container per `go test` package process, reused across
// suites/tests in that process. Each reuse truncates all tables
// (TRUNCATE ... RESTART IDENTITY CASCADE) and flushes Redis, which is the
// data-equivalent of a fresh container at a fraction of the cost. Migrations
// run once per process; seeds re-run per suite via the existing
// RunAllMigrations/RunAllSeeds call sites (upsert-safe: ON CONFLICT).
var (
	sharedPGMu       sync.Mutex
	sharedPG         *postgres.PostgresContainer
	sharedPGDB       *gorm.DB
	sharedPGMigrated bool
	// sharedPGExternal is true when the shared handle dials an external DB
	// (TEST_USE_EXTERNAL=1) instead of owning a Testcontainer.
	sharedPGExternal bool
)

// useExternalTestDB reports whether tests should bypass Testcontainers and
// dial externally-provisioned Postgres/Redis (docker-compose.test.yml).
// Opt-in only: default path remains hermetic Testcontainers.
func useExternalTestDB() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("TEST_USE_EXTERNAL")))
	return v == "1" || v == "true" || v == "yes"
}

// externalPGDSN resolves the Postgres DSN for external mode.
// Priority: TEST_PG_DSN, else TEST_PG_* parts with sane local defaults
// matching docker-compose.test.yml.
func externalPGDSN() string {
	if dsn := strings.TrimSpace(os.Getenv("TEST_PG_DSN")); dsn != "" {
		return dsn
	}
	host := envOr("TEST_PG_HOST", "127.0.0.1")
	port := envOr("TEST_PG_PORT", "5433")
	user := envOr("TEST_PG_USER", "postgres")
	pass := envOr("TEST_PG_PASSWORD", "postgres")
	db := envOr("TEST_PG_DB", "ecommerce_test")
	ssl := envOr("TEST_PG_SSLMODE", "disable")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&TimeZone=UTC",
		user, pass, host, port, db, ssl)
}

// externalRedisAddr resolves a Redis addr for external mode.
func externalRedisAddr(prefix, defPort string) string {
	if addr := strings.TrimSpace(os.Getenv(prefix + "_ADDR")); addr != "" {
		return addr
	}
	host := envOr(prefix+"_HOST", "127.0.0.1")
	port := envOr(prefix+"_PORT", defPort)
	return host + ":" + port
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// openGormWithRetry opens GORM against connStr, retrying while the backend
// accepts TCP. Shared by container-backed and external paths.
func openGormWithRetry(t *testing.T, ctx context.Context, connStr string) *gorm.DB {
	t.Helper()
	var gormDB *gorm.DB
	var err error
	const maxDBAttempts = 15
	// Brief pause after the log-based wait — under load the port can still
	// refuse connections for a moment.
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
				return gormDB
			}
			err = dbErr
		}
		if attempt == maxDBAttempts {
			t.Fatalf("failed to connect to database after %d attempts: %v", maxDBAttempts, err)
		}
		time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
	}
	return gormDB
}

// acquireSharedPG returns the process-wide Postgres handle, starting it on
// first use. On reuse it truncates all tables so the caller observes a clean
// slate equivalent to a fresh container; the caller's existing
// RunAllMigrations/RunAllSeeds calls then repopulate (migrations are skipped
// via the migrated guard, seeds are upsert-safe).
// Returns the container (nil in external mode), the GORM handle, and whether
// the handle was reused.
func acquireSharedPG(t *testing.T, ctx context.Context) (*postgres.PostgresContainer, *gorm.DB, bool) {
	t.Helper()
	sharedPGMu.Lock()
	defer sharedPGMu.Unlock()

	// External mode: single shared *gorm.DB, never a container.
	if useExternalTestDB() {
		if sharedPGDB != nil && sharedPGExternal {
			if err := pingGorm(ctx, sharedPGDB); err == nil {
				if err := truncateAllTables(sharedPGDB); err != nil {
					t.Fatalf("failed to reset external test database: %v", err)
				}
				return nil, sharedPGDB, true
			}
			t.Logf("external test database unreachable; redialing")
			sharedPGDB = nil
		}
		dsn := externalPGDSN()
		sharedPGDB = openGormWithRetry(t, ctx, dsn)
		sharedPGExternal = true
		// Do not trust schema state across runs: always (re)run migrations
		// on first use in this process; RunAllMigrations skips afterwards.
		sharedPGMigrated = false
		return nil, sharedPGDB, false
	}

	if sharedPGDB != nil && sharedPG != nil {
		if err := pingGorm(ctx, sharedPGDB); err == nil {
			if err := truncateAllTables(sharedPGDB); err != nil {
				t.Fatalf("failed to reset shared test database: %v", err)
			}
			return sharedPG, sharedPGDB, true
		}
		t.Logf("shared postgres died; starting a replacement")
		_ = sharedPG.Terminate(ctx)
		sharedPG, sharedPGDB = nil, nil
		sharedPGMigrated = false
	}

	pgContainer, err := startPGContainer(ctx)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable&TimeZone=UTC")
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("failed to get postgres connection string: %v", err)
	}
	sharedPG = pgContainer
	sharedPGDB = openGormWithRetry(t, ctx, connStr)
	sharedPGExternal = false
	sharedPGMigrated = false
	return sharedPG, sharedPGDB, false
}

// startPGContainer boots one postgres:16-alpine Testcontainer.
func startPGContainer(ctx context.Context) (*postgres.PostgresContainer, error) {
	return postgres.Run(ctx,
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
}

func pingGorm(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return sqlDB.PingContext(pingCtx)
}

// isSharedPGMigrated reports whether the shared backend already has the
// schema applied in this process.
func isSharedPGMigrated(db *gorm.DB) bool {
	sharedPGMu.Lock()
	defer sharedPGMu.Unlock()
	return sharedPGMigrated && db != nil && db == sharedPGDB
}

// markSharedPGMigrated records a successful full migration run.
func markSharedPGMigrated(db *gorm.DB) {
	sharedPGMu.Lock()
	defer sharedPGMu.Unlock()
	if db != nil && db == sharedPGDB {
		sharedPGMigrated = true
	}
}

// truncateAllTables removes all rows from every table in the public schema
// and restarts identities. Single statement with CASCADE so FK order does not
// matter. Equivalent to a fresh database for test isolation; schema,
// extensions and (absent) migration bookkeeping are preserved — the repo
// applies raw SQL with no tracking table, so nothing must be excluded.
func truncateAllTables(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}
	var tables []string
	if err := db.Raw(`
		SELECT tablename FROM pg_tables
		WHERE schemaname = 'public'
	`).Scan(&tables).Error; err != nil {
		return err
	}
	if len(tables) == 0 {
		return nil
	}
	quoted := make([]string, 0, len(tables))
	for _, tbl := range tables {
		quoted = append(quoted, `"`+strings.ReplaceAll(tbl, `"`, `""`)+`"`)
	}
	return db.Exec(
		"TRUNCATE TABLE " + strings.Join(quoted, ", ") + " RESTART IDENTITY CASCADE",
	).Error
}

// dialExternalRedis dials an externally-provisioned Redis at addr and waits
// until PING succeeds. Used only with TEST_USE_EXTERNAL=1.
func dialExternalRedis(t *testing.T, ctx context.Context, addr, role string) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
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
			t.Fatalf("%s (external %s) not responding to PING: %v", role, addr, last)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// ResetState clears all Postgres rows and flushes both Redis roles for the
// given container handle. Suites may call it explicitly for mid-suite
// isolation; SetupTestContainers already resets on acquire (reset-before-use,
// which is crash-safe unlike reset-after-test).
func (tc *TestContainer) ResetState(t *testing.T) {
	t.Helper()
	if tc == nil {
		t.Fatalf("ResetState called on nil TestContainer")
	}
	if tc.DB != nil {
		if err := truncateAllTables(tc.DB); err != nil {
			t.Fatalf("failed to reset test database: %v", err)
		}
	}
	ctx := context.Background()
	if tc.RedisClient != nil {
		_ = tc.RedisClient.FlushAll(ctx).Err()
	}
	if tc.DurableKVClient != nil {
		_ = tc.DurableKVClient.FlushAll(ctx).Err()
	}
	ResetAllModuleSingletons()
}
