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

	productEntity "ecommerce-be/product_management/entity"
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

	// Auto-migrate the test schema for product management entities
	err = db.AutoMigrate(
		&productEntity.Category{},
		&productEntity.AttributeDefinition{},
		&productEntity.CategoryAttribute{},
		&productEntity.Product{},
		&productEntity.ProductAttribute{},
		&productEntity.PackageOption{},
	)
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
	// Clean up in reverse order of dependencies
	_ = tc.DB.Exec("TRUNCATE TABLE package_options CASCADE").Error
	_ = tc.DB.Exec("TRUNCATE TABLE package_options CASCADE").Error
	_ = tc.DB.Exec("TRUNCATE TABLE product_attributes CASCADE").Error
	_ = tc.DB.Exec("TRUNCATE TABLE category_attributes CASCADE").Error
	_ = tc.DB.Exec("TRUNCATE TABLE products CASCADE").Error
	_ = tc.DB.Exec("TRUNCATE TABLE attribute_definitions CASCADE").Error
	_ = tc.DB.Exec("TRUNCATE TABLE categories CASCADE").Error

	// For test tables created during tests, we need to identify and clean them
	// Get all user-created tables (excluding system tables)
	var tables []string
	err := tc.DB.Raw(`
		SELECT tablename 
		FROM pg_tables 
		WHERE schemaname = 'public' 
		AND tablename NOT IN ('categories', 'attribute_definitions', 'category_attributes', 'products', 'product_attributes', 'package_options')
	`).Scan(&tables).Error

	if err == nil {
		// Clean additional test tables
		for _, table := range tables {
			_ = tc.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error
		}
	}

	log.Printf("🧹 Database cleaned successfully")
}

// GetDB returns the GORM database instance
func (tc *PostgreSQLTestContainer) GetDB() *gorm.DB {
	return tc.DB
}

// GetConnectionString returns the database connection string
func (tc *PostgreSQLTestContainer) GetConnectionString() string {
	return tc.DSN
}

// IsHealthy checks if the database connection is healthy
func (tc *PostgreSQLTestContainer) IsHealthy() bool {
	if tc.DB == nil {
		return false
	}

	sqlDB, err := tc.DB.DB()
	if err != nil {
		return false
	}

	err = sqlDB.Ping()
	return err == nil
}

// GetTableCount returns the number of records in a specific table
func (tc *PostgreSQLTestContainer) GetTableCount(t *testing.T, tableName string) int64 {
	var count int64
	err := tc.DB.Table(tableName).Count(&count).Error
	require.NoError(t, err, "Failed to get table count for %s", tableName)
	return count
}

// VerifyTableExists checks if a table exists in the database
func (tc *PostgreSQLTestContainer) VerifyTableExists(t *testing.T, tableName string) bool {
	var exists bool
	err := tc.DB.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = ?
		)
	)
	`, tableName).Scan(&exists).Error

	require.NoError(t, err, "Failed to check if table %s exists", tableName)
	return exists
}

// GetTableSchema returns the schema information for a table
func (tc *PostgreSQLTestContainer) GetTableSchema(t *testing.T, tableName string) []struct {
	ColumnName    string `json:"column_name"`
	DataType      string `json:"data_type"`
	IsNullable    string `json:"is_nullable"`
	ColumnDefault string `json:"column_default"`
} {
	var columns []struct {
		ColumnName    string `json:"column_name"`
		DataType      string `json:"data_type"`
		IsNullable    string `json:"is_nullable"`
		ColumnDefault string `json:"column_default"`
	}

	err := tc.DB.Raw(`
		SELECT 
			column_name,
			data_type,
			is_nullable,
			COALESCE(column_default, '') as column_default
		FROM information_schema.columns 
		WHERE table_schema = 'public' 
		AND table_name = ?
		ORDER BY ordinal_position
	`, tableName).Scan(&columns).Error

	require.NoError(t, err, "Failed to get table schema for %s", tableName)
	return columns
}

// ExecuteRawSQL executes raw SQL and returns the result
func (tc *PostgreSQLTestContainer) ExecuteRawSQL(t *testing.T, sql string, args ...interface{}) *gorm.DB {
	result := tc.DB.Exec(sql, args...)
	require.NoError(t, result.Error, "Failed to execute raw SQL: %s", sql)
	return result
}

// BeginTransaction starts a new database transaction
func (tc *PostgreSQLTestContainer) BeginTransaction() *gorm.DB {
	return tc.DB.Begin()
}

// RollbackTransaction rolls back a database transaction
func (tc *PostgreSQLTestContainer) RollbackTransaction(tx *gorm.DB) {
	if tx != nil {
		tx.Rollback()
	}
}

// CommitTransaction commits a database transaction
func (tc *PostgreSQLTestContainer) CommitTransaction(tx *gorm.DB) error {
	if tx != nil {
		return tx.Commit().Error
	}
	return nil
}

// WaitForDatabase waits for the database to be ready
func (tc *PostgreSQLTestContainer) WaitForDatabase(t *testing.T, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Database not ready within timeout")
		case <-ticker.C:
			if tc.IsHealthy() {
				log.Printf("✅ Database is ready")
				return
			}
		}
	}
}

// GetDatabaseInfo returns basic database information
func (tc *PostgreSQLTestContainer) GetDatabaseInfo(t *testing.T) map[string]interface{} {
	var version string
	err := tc.DB.Raw("SELECT version()").Scan(&version).Error
	require.NoError(t, err, "Failed to get database version")

	var currentDB string
	err = tc.DB.Raw("SELECT current_database()").Scan(&currentDB).Error
	require.NoError(t, err, "Failed to get current database")

	var currentUser string
	err = tc.DB.Raw("SELECT current_user").Scan(&currentUser).Error
	require.NoError(t, err, "Failed to get current user")

	return map[string]interface{}{
		"version":    version,
		"database":   currentDB,
		"user":       currentUser,
		"connection": tc.DSN,
		"is_healthy": tc.IsHealthy(),
	}
}
