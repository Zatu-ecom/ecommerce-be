package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostgreSQLContainer tests the PostgreSQL container lifecycle
func TestPostgreSQLContainer(t *testing.T) {
	t.Log("🧪 Testing PostgreSQL container lifecycle...")

	// Setup container
	container := SetupPostgreSQLContainer(t)
	require.NotNil(t, container, "Container should not be nil")
	defer container.Cleanup(t)

	// Test container is healthy
	t.Run("Container Health Check", func(t *testing.T) {
		assert.True(t, container.IsHealthy(), "Container should be healthy")
	})

	// Test database connection
	t.Run("Database Connection", func(t *testing.T) {
		db := container.GetDB()
		require.NotNil(t, db, "Database should not be nil")

		// Test basic query
		var result int
		err := db.Raw("SELECT 1").Scan(&result).Error
		require.NoError(t, err, "Should be able to execute basic query")
		assert.Equal(t, 1, result, "Query result should be 1")
	})

	// Test database info
	t.Run("Database Information", func(t *testing.T) {
		info := container.GetDatabaseInfo(t)
		require.NotNil(t, info, "Database info should not be nil")

		assert.NotEmpty(t, info["version"], "Database version should not be empty")
		assert.Equal(t, "testdb", info["database"], "Database name should be testdb")
		assert.Equal(t, "testuser", info["user"], "Database user should be testuser")
		assert.True(t, info["is_healthy"].(bool), "Database should be healthy")
	})

	// Test table creation and verification
	t.Run("Table Management", func(t *testing.T) {
		// Verify required tables exist
		requiredTables := []string{
			"categories",
			"attribute_definitions",
			"category_attributes",
			"products",
			"product_attributes",
			"package_options",
		}

		for _, tableName := range requiredTables {
			exists := container.VerifyTableExists(t, tableName)
			assert.True(t, exists, "Table %s should exist", tableName)
		}

		// Test table schema
		schema := container.GetTableSchema(t, "categories")
		require.NotEmpty(t, schema, "Categories table schema should not be empty")

		// Verify key columns exist
		columnNames := make(map[string]bool)
		for _, col := range schema {
			columnNames[col.ColumnName] = true
		}

		expectedColumns := []string{"id", "name", "description", "parent_id", "is_active", "created_at", "updated_at"}
		for _, expectedCol := range expectedColumns {
			assert.True(t, columnNames[expectedCol], "Column %s should exist in categories table", expectedCol)
		}
	})

	// Test database cleanup
	t.Run("Database Cleanup", func(t *testing.T) {
		// Create some test data first
		container.CleanDatabase(t)

		// Verify tables are empty
		expectedTables := []string{
			"categories",
			"attribute_definitions",
			"category_attributes",
			"products",
			"product_attributes",
			"package_options",
		}

		for _, tableName := range expectedTables {
			count := container.GetTableCount(t, tableName)
			assert.Equal(t, int64(0), count, "Table %s should be empty after cleanup", tableName)
		}
	})

	// Test transaction management
	t.Run("Transaction Management", func(t *testing.T) {
		tx := container.BeginTransaction()
		require.NotNil(t, tx, "Transaction should not be nil")

		// Execute a query within transaction
		var result int
		err := tx.Raw("SELECT 42").Scan(&result).Error
		require.NoError(t, err, "Should be able to execute query within transaction")
		assert.Equal(t, 42, result, "Transaction query result should be 42")

		// Rollback transaction
		container.RollbackTransaction(tx)
	})

	// Test raw SQL execution
	t.Run("Raw SQL Execution", func(t *testing.T) {
		// Create a test table
		sql := `CREATE TABLE IF NOT EXISTS test_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`

		result := container.ExecuteRawSQL(t, sql)
		require.NoError(t, result.Error, "Should be able to execute raw SQL")

		// Verify table was created
		exists := container.VerifyTableExists(t, "test_table")
		assert.True(t, exists, "Test table should be created")

		// Clean up test table
		container.ExecuteRawSQL(t, "DROP TABLE test_table")
	})

	// Test database readiness
	t.Run("Database Readiness", func(t *testing.T) {
		// Test with a short timeout
		container.WaitForDatabase(t, 5*time.Second)
		assert.True(t, container.IsHealthy(), "Database should be ready after wait")
	})

	t.Log("✅ PostgreSQL container tests completed successfully")
}

// TestContainerResilience tests container resilience and error handling
func TestContainerResilience(t *testing.T) {
	t.Log("🧪 Testing container resilience...")

	// Test multiple container instances
	t.Run("Multiple Container Instances", func(t *testing.T) {
		containers := make([]*PostgreSQLTestContainer, 3)

		// Create multiple containers
		for i := 0; i < 3; i++ {
			container := SetupPostgreSQLContainer(t)
			require.NotNil(t, container, "Container %d should not be nil", i)
			containers[i] = container
		}

		// Verify all containers are healthy
		for i, container := range containers {
			assert.True(t, container.IsHealthy(), "Container %d should be healthy", i)
		}

		// Cleanup all containers
		for _, container := range containers {
			container.Cleanup(t)
		}
	})

	// Test container restart simulation
	t.Run("Container Restart Simulation", func(t *testing.T) {
		container := SetupPostgreSQLContainer(t)
		require.NotNil(t, container, "Container should not be nil")
		defer container.Cleanup(t)

		// Simulate container restart by cleaning up and recreating connection
		container.CleanDatabase(t)

		// Verify container is still healthy after cleanup
		assert.True(t, container.IsHealthy(), "Container should remain healthy after cleanup")
	})

	t.Log("✅ Container resilience tests completed successfully")
}

// TestContainerPerformance tests container performance characteristics
func TestContainerPerformance(t *testing.T) {
	t.Log("🧪 Testing container performance...")

	container := SetupPostgreSQLContainer(t)
	require.NotNil(t, container, "Container should not be nil")
	defer container.Cleanup(t)

	t.Run("Connection Speed", func(t *testing.T) {
		start := time.Now()

		// Perform multiple database operations
		for i := 0; i < 100; i++ {
			var result int
			err := container.GetDB().Raw("SELECT ?", i).Scan(&result).Error
			require.NoError(t, err, "Query %d should succeed", i)
			assert.Equal(t, i, result, "Query %d result should match", i)
		}

		duration := time.Since(start)
		t.Logf("100 queries completed in %v", duration)

		// Performance assertion (adjust based on your system)
		assert.Less(t, duration, 2*time.Second, "100 queries should complete within 2 seconds")
	})

	t.Run("Bulk Operations", func(t *testing.T) {
		start := time.Now()

		// Create test table for bulk operations
		container.ExecuteRawSQL(t, `
			CREATE TABLE IF NOT EXISTS bulk_test (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100),
				value INTEGER
			)
		`)

		// Insert bulk data
		for i := 0; i < 1000; i++ {
			container.ExecuteRawSQL(t,
				"INSERT INTO bulk_test (name, value) VALUES ($1, $2)",
				fmt.Sprintf("item_%d", i), i,
			)
		}

		// Verify bulk data
		count := container.GetTableCount(t, "bulk_test")
		assert.Equal(t, int64(1000), count, "Should have 1000 records")

		// Cleanup
		container.ExecuteRawSQL(t, "DROP TABLE bulk_test")

		duration := time.Since(start)
		t.Logf("Bulk operations completed in %v", duration)

		// Performance assertion
		assert.Less(t, duration, 5*time.Second, "Bulk operations should complete within 5 seconds")
	})

	t.Log("✅ Container performance tests completed successfully")
}

// TestContainerIsolation tests container isolation and data separation
func TestContainerIsolation(t *testing.T) {
	t.Log("🧪 Testing container isolation...")

	// Create two separate containers
	container1 := SetupPostgreSQLContainer(t)
	require.NotNil(t, container1, "Container 1 should not be nil")
	defer container1.Cleanup(t)

	container2 := SetupPostgreSQLContainer(t)
	require.NotNil(t, container2, "Container 2 should not be nil")
	defer container2.Cleanup(t)

	t.Run("Data Isolation", func(t *testing.T) {
		// Create test data in container 1
		container1.ExecuteRawSQL(t, `
			CREATE TABLE IF NOT EXISTS isolation_test (
				id SERIAL PRIMARY KEY,
				data VARCHAR(100)
			)
		`)
		container1.ExecuteRawSQL(t,
			"INSERT INTO isolation_test (data) VALUES ($1)",
			"container1_data",
		)

		// Verify data exists in container 1
		count1 := container1.GetTableCount(t, "isolation_test")
		assert.Equal(t, int64(1), count1, "Container 1 should have 1 record")

		// Verify data does not exist in container 2
		exists2 := container2.VerifyTableExists(t, "isolation_test")
		assert.False(t, exists2, "Container 2 should not have isolation_test table")

		// Cleanup
		container1.ExecuteRawSQL(t, "DROP TABLE isolation_test")
	})

	t.Run("Connection String Isolation", func(t *testing.T) {
		connStr1 := container1.GetConnectionString()
		connStr2 := container2.GetConnectionString()

		// Connection strings should be different (different ports)
		assert.NotEqual(t, connStr1, connStr2, "Connection strings should be different")
	})

	t.Log("✅ Container isolation tests completed successfully")
}
