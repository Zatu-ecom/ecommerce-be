package tests

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"datun.com/be/user_management/entity"
)

// PostgreSQLTestContainer wraps the PostgreSQL test container
type PostgreSQLTestContainer struct {
	Container testcontainers.Container
	DB        *gorm.DB
	DSN       string
	ctx       context.Context
}

// SetupPostgreSQLContainer creates and starts a PostgreSQL test container
func SetupPostgreSQLContainer(t *testing.T) *PostgreSQLTestContainer {
	ctx := context.Background()

	// Create PostgreSQL container
	postgresContainer, err := postgrescontainer.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgrescontainer.WithDatabase("testdb"),
		postgrescontainer.WithUsername("testuser"),
		postgrescontainer.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Get the connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "Failed to get connection string")

	// Connect to the database with GORM
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "Failed to connect to test database")

	// Auto-migrate the test schema
	err = db.AutoMigrate(&entity.User{}, &entity.Address{})
	require.NoError(t, err, "Failed to migrate test database")

	// Log successful setup
	log.Printf("✅ PostgreSQL test container started successfully")
	log.Printf("📊 Connection string: %s", connStr)

	return &PostgreSQLTestContainer{
		Container: postgresContainer,
		DB:        db,
		DSN:       connStr,
		ctx:       ctx,
	}
}

// Cleanup terminates the PostgreSQL test container
func (tc *PostgreSQLTestContainer) Cleanup(t *testing.T) {
	if tc.Container != nil {
		err := tc.Container.Terminate(tc.ctx)
		if err != nil {
			t.Logf("Failed to terminate PostgreSQL container: %v", err)
		} else {
			log.Printf("🛑 PostgreSQL test container terminated successfully")
		}
	}
}

// CleanDatabase cleans all data from the test database
func (tc *PostgreSQLTestContainer) CleanDatabase(t *testing.T) {
	// First, clean entity tables (if they exist)
	_ = tc.DB.Exec("TRUNCATE TABLE addresses CASCADE").Error // Ignore errors if table doesn't exist
	_ = tc.DB.Exec("TRUNCATE TABLE users CASCADE").Error     // Ignore errors if table doesn't exist

	// For test tables created during tests, we need to identify and clean them
	// Get all user-created tables (excluding system tables)
	var tables []string
	err := tc.DB.Raw(`
		SELECT tablename 
		FROM pg_tables 
		WHERE schemaname = 'public' 
		AND tablename NOT IN ('users', 'addresses')
	`).Scan(&tables).Error

	if err == nil {
		// Clean additional test tables
		for _, table := range tables {
			_ = tc.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error
		}
	}
}

// GetConnectionInfo returns database connection information for tests
func (tc *PostgreSQLTestContainer) GetConnectionInfo() (string, *gorm.DB) {
	return tc.DSN, tc.DB
}

// HealthCheck verifies that the database connection is healthy
func (tc *PostgreSQLTestContainer) HealthCheck(t *testing.T) {
	sqlDB, err := tc.DB.DB()
	require.NoError(t, err, "Failed to get underlying SQL DB")

	err = sqlDB.Ping()
	require.NoError(t, err, "Database health check failed")

	// Test a simple query
	var result int
	err = tc.DB.Raw("SELECT 1").Scan(&result).Error
	require.NoError(t, err, "Failed to execute test query")
	require.Equal(t, 1, result, "Test query returned unexpected result")
}

// WaitForReady waits for the database to be ready for connections
func (tc *PostgreSQLTestContainer) WaitForReady(t *testing.T, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(tc.ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for database to be ready")
		case <-ticker.C:
			if sqlDB, err := tc.DB.DB(); err == nil {
				if err := sqlDB.Ping(); err == nil {
					log.Printf("✅ Database is ready for connections")
					return
				}
			}
		}
	}
}

// GetTestMetrics returns useful metrics about the test container
func (tc *PostgreSQLTestContainer) GetTestMetrics(t *testing.T) map[string]interface{} {
	metrics := make(map[string]interface{})

	// Get container ID
	metrics["container_id"] = tc.Container.GetContainerID()

	// Get exposed ports
	if mappedPort, err := tc.Container.MappedPort(tc.ctx, "5432"); err == nil {
		metrics["mapped_port"] = mappedPort.Port()
	}

	// Get database connection stats
	if sqlDB, err := tc.DB.DB(); err == nil {
		stats := sqlDB.Stats()
		metrics["open_connections"] = stats.OpenConnections
		metrics["in_use"] = stats.InUse
		metrics["idle"] = stats.Idle
	}

	return metrics
}
