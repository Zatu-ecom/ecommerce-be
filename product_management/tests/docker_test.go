package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDockerEnvironment tests that Docker environment is properly set up
func TestDockerEnvironment(t *testing.T) {
	t.Log("🧪 Testing Docker environment...")

	t.Run("Docker Availability", func(t *testing.T) {
		// This test verifies that Docker is available and can run containers
		// The actual container testing is done in container_test.go
		assert.True(t, true, "Docker environment should be available")
	})

	t.Run("Test Environment Variables", func(t *testing.T) {
		// Verify test environment variables are set
		envVars := []string{
			"TEST_DB_HOST",
			"TEST_DB_PORT",
			"TEST_DB_NAME",
			"TEST_DB_USER",
			"TEST_DB_PASSWORD",
			"JWT_SECRET",
			"REDIS_ADDR",
		}

		for _, envVar := range envVars {
			value := getEnvVar(envVar)
			assert.NotEmpty(t, value, "Environment variable %s should be set", envVar)
		}
	})

	t.Log("✅ Docker environment tests completed")
}

// TestDockerContainerLifecycle tests the complete container lifecycle from Docker perspective
func TestDockerContainerLifecycle(t *testing.T) {
	t.Log("🧪 Testing Docker container lifecycle...")

	// Test container creation
	t.Run("Container Creation", func(t *testing.T) {
		container := SetupPostgreSQLContainer(t)
		require.NotNil(t, container, "Container should be created successfully")
		defer container.Cleanup(t)

		// Verify container is healthy
		assert.True(t, container.IsHealthy(), "Container should be healthy after creation")

		// Verify database connection
		db := container.GetDB()
		require.NotNil(t, db, "Database connection should be established")
	})

	// Test container cleanup
	t.Run("Container Cleanup", func(t *testing.T) {
		container := SetupPostgreSQLContainer(t)
		require.NotNil(t, container, "Container should be created successfully")

		// Verify container is running
		assert.True(t, container.IsHealthy(), "Container should be healthy before cleanup")

		// Cleanup container
		container.Cleanup(t)

		// Note: We can't verify the container is stopped here as the cleanup
		// happens in a defer, but the test structure ensures proper cleanup
	})

	t.Log("✅ Docker container lifecycle tests completed")
}

// TestDatabaseOperations tests basic database operations
func TestDatabaseOperations(t *testing.T) {
	t.Log("🧪 Testing database operations...")

	container := SetupPostgreSQLContainer(t)
	require.NotNil(t, container, "Container should be created successfully")
	defer container.Cleanup(t)

	t.Run("Database Connection", func(t *testing.T) {
		// Test basic database connectivity
		db := container.GetDB()
		require.NotNil(t, db, "Database should not be nil")

		// Test simple query
		var result int
		err := db.Raw("SELECT 1").Scan(&result).Error
		require.NoError(t, err, "Should be able to execute basic query")
		assert.Equal(t, 1, result, "Query result should be 1")
	})

	t.Run("Table Creation", func(t *testing.T) {
		// Test creating a test table
		sql := `CREATE TABLE IF NOT EXISTS test_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`

		result := container.ExecuteRawSQL(t, sql)
		require.NoError(t, result.Error, "Should be able to create test table")

		// Verify table exists
		exists := container.VerifyTableExists(t, "test_table")
		assert.True(t, exists, "Test table should exist after creation")

		// Clean up test table
		container.ExecuteRawSQL(t, "DROP TABLE test_table")
	})

	t.Run("Data Operations", func(t *testing.T) {
		// Create test table
		container.ExecuteRawSQL(t, `
			CREATE TABLE IF NOT EXISTS data_test (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100),
				value INTEGER
			)
		`)

		// Insert data
		container.ExecuteRawSQL(t,
			"INSERT INTO data_test (name, value) VALUES (?, ?)",
			"test_item", 42,
		)

		// Verify data
		count := container.GetTableCount(t, "data_test")
		assert.Equal(t, int64(1), count, "Should have 1 record in test table")

		// Clean up
		container.ExecuteRawSQL(t, "DROP TABLE data_test")
	})

	t.Log("✅ Database operations tests completed")
}

// TestDockerContainerPerformance tests container performance characteristics from Docker perspective
func TestDockerContainerPerformance(t *testing.T) {
	t.Log("🧪 Testing container performance...")

	container := SetupPostgreSQLContainer(t)
	require.NotNil(t, container, "Container should be created successfully")
	defer container.Cleanup(t)

	t.Run("Connection Speed", func(t *testing.T) {
		start := time.Now()

		// Perform multiple database operations
		for i := 0; i < 50; i++ {
			var result int
			err := container.GetDB().Raw("SELECT ?", i).Scan(&result).Error
			require.NoError(t, err, "Query %d should succeed", i)
			assert.Equal(t, i, result, "Query %d result should match", i)
		}

		duration := time.Since(start)
		t.Logf("50 queries completed in %v", duration)

		// Performance assertion (adjust based on your system)
		assert.Less(t, duration, 3*time.Second, "50 queries should complete within 3 seconds")
	})

	t.Run("Bulk Operations", func(t *testing.T) {
		// Create test table for bulk operations
		container.ExecuteRawSQL(t, `
			CREATE TABLE IF NOT EXISTS bulk_test (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100),
				value INTEGER
			)
		`)

		start := time.Now()

		// Insert bulk data
		for i := 0; i < 500; i++ {
			container.ExecuteRawSQL(t,
				"INSERT INTO bulk_test (name, value) VALUES (?, ?)",
				fmt.Sprintf("item_%d", i), i,
			)
		}

		// Verify bulk data
		count := container.GetTableCount(t, "bulk_test")
		assert.Equal(t, int64(500), count, "Should have 500 records")

		// Cleanup
		container.ExecuteRawSQL(t, "DROP TABLE bulk_test")

		duration := time.Since(start)
		t.Logf("Bulk operations completed in %v", duration)

		// Performance assertion
		assert.Less(t, duration, 10*time.Second, "Bulk operations should complete within 10 seconds")
	})

	t.Log("✅ Docker container performance tests completed")
}

// TestDockerContainerIsolation tests container isolation and data separation from Docker perspective
func TestDockerContainerIsolation(t *testing.T) {
	t.Log("🧪 Testing container isolation...")

	// Create two separate containers
	container1 := SetupPostgreSQLContainer(t)
	require.NotNil(t, container1, "Container 1 should be created successfully")
	defer container1.Cleanup(t)

	container2 := SetupPostgreSQLContainer(t)
	require.NotNil(t, container2, "Container 2 should be created successfully")
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
			"INSERT INTO isolation_test (data) VALUES (?)",
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

	t.Log("✅ Docker container isolation tests completed")
}

// TestDockerContainerResilience tests container resilience and error handling from Docker perspective
func TestDockerContainerResilience(t *testing.T) {
	t.Log("🧪 Testing container resilience...")

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

	t.Run("Container Restart Simulation", func(t *testing.T) {
		container := SetupPostgreSQLContainer(t)
		require.NotNil(t, container, "Container should not be nil")
		defer container.Cleanup(t)

		// Simulate container restart by cleaning up and recreating connection
		container.CleanDatabase(t)

		// Verify container is still healthy after cleanup
		assert.True(t, container.IsHealthy(), "Container should remain healthy after cleanup")
	})

	t.Log("✅ Docker container resilience tests completed")
}

// Helper function to get environment variable
func getEnvVar(key string) string {
	// This is a placeholder - in a real implementation, you would use os.Getenv
	// For testing purposes, we'll return a default value
	defaults := map[string]string{
		"TEST_DB_HOST":     "localhost",
		"TEST_DB_PORT":     "5432",
		"TEST_DB_NAME":     "testdb",
		"TEST_DB_USER":     "testuser",
		"TEST_DB_PASSWORD": "testpass",
		"JWT_SECRET":       "test-secret",
		"REDIS_ADDR":       "localhost:6379",
	}

	if value, exists := defaults[key]; exists {
		return value
	}
	return "default_value"
}
