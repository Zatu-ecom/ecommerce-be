package setup

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TestContainers holds the containers for testing
type TestContainers struct {
	Postgres    *postgres.PostgresContainer
	Redis       testcontainers.Container
	DB          *gorm.DB
	RedisClient *redis.Client
	ctx         context.Context
}

// SetupTestContainers sets up the test containers for Postgres and Redis
func SetupTestContainers(t *testing.T) *TestContainers {
	ctx := context.Background()

	// Postgres container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// Redis container
	redisContainer, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "redis:7-alpine",
				ExposedPorts: []string{"6379/tcp"},
				WaitingFor:   wait.ForLog("Ready to accept connections"),
			},
			Started: true,
		},
	)
	if err != nil {
		t.Fatalf("failed to start redis container: %v", err)
	}

	// Get Postgres connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable&TimeZone=UTC")
	if err != nil {
		t.Fatalf("failed to get postgres connection string: %v", err)
	}

	// Connect to the database with GORM
	// Use the same configuration as production (singular table names)
	gormDB, err := gorm.Open(gormpostgres.Open(connStr), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // Use singular table names to match production
		},
	})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Get Redis connection details
	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get redis host: %v", err)
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get redis port: %v", err)
	}

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
	})

	return &TestContainers{
		Postgres:    pgContainer,
		Redis:       redisContainer,
		DB:          gormDB,
		RedisClient: redisClient,
		ctx:         ctx,
	}
}

// Cleanup terminates the test containers
func (tc *TestContainers) Cleanup(t *testing.T) {
	if err := tc.Postgres.Terminate(tc.ctx); err != nil {
		t.Logf("failed to terminate postgres container: %v", err)
	}
	if err := tc.Redis.Terminate(tc.ctx); err != nil {
		t.Logf("failed to terminate redis container: %v", err)
	}
}
