package tests

import (
	"os"
	"testing"
)

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Setup code before running tests
	setup()

	// Run tests
	code := m.Run()

	// Cleanup code after running tests
	teardown()

	// Exit with the test result code
	os.Exit(code)
}

func setup() {
	// Initialize test environment
	// This could include:
	// - Setting up test database
	// - Initializing Redis
	// - Loading test configuration
	// - Setting environment variables for testing

	os.Setenv("GIN_MODE", "test")
	os.Setenv("JWT_SECRET", "test-jwt-secret-key")
	os.Setenv("REDIS_URL", "localhost:6379")
	os.Setenv("REDIS_PASSWORD", "")
	os.Setenv("REDIS_DB", "1") // Use different Redis DB for tests
}

func teardown() {
	// Clean up test environment
	// This could include:
	// - Cleaning up test data
	// - Closing database connections
	// - Resetting environment variables
}
