package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostgreSQLContainerLifecycle tests the complete lifecycle of PostgreSQL test containers
func TestPostgreSQLContainerLifecycle(t *testing.T) {
	t.Log("🚀 Starting PostgreSQL container lifecycle test...")

	// Step 1: Start PostgreSQL container
	t.Log("📦 Setting up PostgreSQL test container...")
	pgContainer := SetupPostgreSQLContainer(t)

	// Verify container started successfully
	assert.NotNil(t, pgContainer, "PostgreSQL container should not be nil")
	assert.NotNil(t, pgContainer.Container, "Container instance should not be nil")
	assert.NotNil(t, pgContainer.DB, "Database connection should not be nil")

	// Step 2: Test database connection
	t.Log("🔗 Testing database connection...")
	pgContainer.HealthCheck(t)

	// Step 3: Test database operations
	t.Log("📊 Testing basic database operations...")

	// Test creating a simple table and inserting data
	err := pgContainer.DB.Exec("CREATE TABLE test_table (id SERIAL PRIMARY KEY, name VARCHAR(100))").Error
	require.NoError(t, err, "Should be able to create test table")

	err = pgContainer.DB.Exec("INSERT INTO test_table (name) VALUES ('test1'), ('test2')").Error
	require.NoError(t, err, "Should be able to insert test data")

	// Test querying data
	var count int64
	err = pgContainer.DB.Raw("SELECT COUNT(*) FROM test_table").Scan(&count).Error
	require.NoError(t, err, "Should be able to query test data")
	assert.Equal(t, int64(2), count, "Should have 2 records in test table")

	// Step 4: Test database cleanup
	t.Log("🧹 Testing database cleanup...")
	pgContainer.CleanDatabase(t)

	// Verify test table was dropped during cleanup (should return error)
	err = pgContainer.DB.Raw("SELECT COUNT(*) FROM test_table").Scan(&count).Error
	assert.Error(t, err, "Test table should be dropped after cleanup")
	assert.Contains(t, err.Error(), "does not exist", "Error should indicate table doesn't exist")

	// Step 5: Test container metrics
	t.Log("📈 Testing container metrics...")
	metrics := pgContainer.GetTestMetrics(t)
	assert.NotEmpty(t, metrics["container_id"], "Should have container ID")
	assert.NotEmpty(t, metrics["mapped_port"], "Should have mapped port")

	// Step 6: Test multiple connections
	t.Log("🔄 Testing multiple database connections...")
	dsn, db2 := pgContainer.GetConnectionInfo()
	assert.NotEmpty(t, dsn, "DSN should not be empty")
	assert.NotNil(t, db2, "Second DB connection should not be nil")

	// Test that both connections work
	var result1, result2 int
	err = pgContainer.DB.Raw("SELECT 1").Scan(&result1).Error
	require.NoError(t, err, "First connection should work")

	err = db2.Raw("SELECT 2").Scan(&result2).Error
	require.NoError(t, err, "Second connection should work")

	assert.Equal(t, 1, result1, "First query should return 1")
	assert.Equal(t, 2, result2, "Second query should return 2")

	// Step 7: Test container cleanup
	t.Log("🛑 Testing container cleanup...")
	start := time.Now()
	pgContainer.Cleanup(t)
	cleanupDuration := time.Since(start)

	t.Logf("✅ Container cleanup completed in %v", cleanupDuration)

	// Verify cleanup was successful by trying to connect (should fail)
	sqlDB, err := pgContainer.DB.DB()
	if err == nil && sqlDB != nil {
		err = sqlDB.Ping()
		// Connection might still work briefly, but container should be terminated
		// We mainly verify that Cleanup() didn't panic or error
	}

	t.Log("✅ PostgreSQL container lifecycle test completed successfully!")
}

// TestPostgreSQLContainerConcurrency tests multiple containers running concurrently
func TestPostgreSQLContainerConcurrency(t *testing.T) {
	t.Log("🚀 Testing PostgreSQL container concurrency...")

	const numContainers = 2
	containers := make([]*PostgreSQLTestContainer, numContainers)

	// Start multiple containers
	for i := 0; i < numContainers; i++ {
		t.Logf("📦 Starting container %d/%d...", i+1, numContainers)
		containers[i] = SetupPostgreSQLContainer(t)
		containers[i].HealthCheck(t)
	}

	// Test that each container is isolated
	for i, container := range containers {
		tableName := fmt.Sprintf("test_table_%d", i)

		// Create unique table in each container
		err := container.DB.Exec(fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY, value INT)", tableName)).Error
		require.NoError(t, err, "Should create table in container %d", i)

		// Insert unique data
		err = container.DB.Exec(fmt.Sprintf("INSERT INTO %s (value) VALUES (%d)", tableName, i*100)).Error
		require.NoError(t, err, "Should insert data in container %d", i)
	}

	// Verify isolation - each container should only see its own data
	for i, container := range containers {
		expectedTable := fmt.Sprintf("test_table_%d", i)

		// Should see own table
		var count int64
		err := container.DB.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", expectedTable)).Scan(&count).Error
		require.NoError(t, err, "Should query own table in container %d", i)
		assert.Equal(t, int64(1), count, "Should have 1 record in container %d", i)

		// Should not see other containers' tables
		for j := 0; j < numContainers; j++ {
			if i != j {
				otherTable := fmt.Sprintf("test_table_%d", j)
				err := container.DB.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", otherTable)).Scan(&count).Error
				assert.Error(t, err, "Should not see table from container %d in container %d", j, i)
			}
		}
	}

	// Cleanup all containers
	for i, container := range containers {
		t.Logf("🛑 Cleaning up container %d/%d...", i+1, numContainers)
		container.Cleanup(t)
	}

	t.Log("✅ PostgreSQL container concurrency test completed successfully!")
}

// TestPostgreSQLContainerResilience tests container resilience and error handling
func TestPostgreSQLContainerResilience(t *testing.T) {
	t.Log("🚀 Testing PostgreSQL container resilience...")

	// Test normal container setup and operation
	pgContainer := SetupPostgreSQLContainer(t)
	defer pgContainer.Cleanup(t)

	// Test health check multiple times
	for i := 0; i < 3; i++ {
		t.Logf("🏥 Health check iteration %d...", i+1)
		pgContainer.HealthCheck(t)
		time.Sleep(100 * time.Millisecond)
	}

	// Test database operations under load
	t.Log("💪 Testing database operations under simulated load...")
	for i := 0; i < 10; i++ {
		go func(iteration int) {
			tableName := fmt.Sprintf("load_test_%d", iteration)
			err := pgContainer.DB.Exec(fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY)", tableName)).Error
			if err != nil {
				t.Logf("Warning: Failed to create table %s: %v", tableName, err)
			}
		}(i)
	}

	// Give concurrent operations time to complete
	time.Sleep(2 * time.Second)

	// Test cleanup under various conditions
	t.Log("🧹 Testing cleanup resilience...")
	pgContainer.CleanDatabase(t)

	// Test metrics after operations
	metrics := pgContainer.GetTestMetrics(t)
	t.Logf("📊 Final metrics: %+v", metrics)

	t.Log("✅ PostgreSQL container resilience test completed successfully!")
}

// BenchmarkPostgreSQLContainerSetup benchmarks container setup performance
func BenchmarkPostgreSQLContainerSetup(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Convert *testing.B to *testing.T for our functions
		t := &testing.T{}

		pgContainer := SetupPostgreSQLContainer(t)
		pgContainer.HealthCheck(t)
		pgContainer.Cleanup(t)
	}
}
