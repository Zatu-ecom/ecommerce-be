package helpers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"ecommerce-be/common/cache"
	"ecommerce-be/common/db"
	"ecommerce-be/product"
	"ecommerce-be/user"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
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
		log.Printf("failed to terminate redis container: %v", err)
	}
}

func getAbsolutePath(relativePath string) (string, error) {
	_, b, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get caller information")
	}
	basepath := filepath.Dir(b)
	return filepath.Join(basepath, "../..", relativePath), nil
}

// RunMigrations runs the SQL migration scripts
func (tc *TestContainers) RunMigrations(t *testing.T, migrationPath string) {
	sqlDB, err := tc.DB.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB from gorm.DB: %v", err)
	}
	runSQLFile(t, sqlDB, migrationPath)
}

// RunSeeds runs the SQL seed scripts
func (tc *TestContainers) RunSeeds(t *testing.T, seedPath string) {
	sqlDB, err := tc.DB.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB from gorm.DB: %v", err)
	}
	runSQLFile(t, sqlDB, seedPath)
}

func runSQLFile(t *testing.T, sqlDB *sql.DB, relativePath string) {
	absPath, err := getAbsolutePath(relativePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path for %s: %v", relativePath, err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", absPath, err)
	}

	sqlStatements := string(content)
	sqlStatements = strings.TrimSpace(sqlStatements)
	if sqlStatements == "" {
		t.Logf("SQL file %s is empty, skipping", absPath)
		return
	}

	// Execute the entire SQL content at once
	if _, err := sqlDB.Exec(sqlStatements); err != nil {
		t.Fatalf("Failed to execute SQL file %s: %v\nSQL content:\n%s", absPath, err, sqlStatements)
	}
}

// SetupTestServer initializes the application for testing
func SetupTestServer(t *testing.T, database *gorm.DB, redisClient *redis.Client) http.Handler {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Set the test database as the global database instance
	// This is required because the modules use db.GetDB() to get the database instance
	if database != nil {
		db.SetDB(database)
	}

	if redisClient != nil {
		cache.SetRedisClient(redisClient)
	}

	// Register modules
	_ = user.NewContainer(router)
	_ = product.NewContainer(router)

	return router
}
