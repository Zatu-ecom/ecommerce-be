package tests

import (
	"log"
	"os"
	"testing"
)

// TestMain sets up the test environment and runs all tests
func TestMain(m *testing.M) {
	// Set test environment variables if not already set
	if os.Getenv("TEST_DB_HOST") == "" {
		os.Setenv("TEST_DB_HOST", "localhost")
	}
	if os.Getenv("TEST_DB_PORT") == "" {
		os.Setenv("TEST_DB_PORT", "5432")
	}
	if os.Getenv("TEST_DB_NAME") == "" {
		os.Setenv("TEST_DB_NAME", "testdb")
	}
	if os.Getenv("TEST_DB_USER") == "" {
		os.Setenv("TEST_DB_USER", "testuser")
	}
	if os.Getenv("TEST_DB_PASSWORD") == "" {
		os.Setenv("TEST_DB_PASSWORD", "testpass")
	}

	// Set JWT secret for testing
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "test-secret-key-for-testing-only")
	}

	// Set Redis configuration for testing
	if os.Getenv("REDIS_ADDR") == "" {
		os.Setenv("REDIS_ADDR", "localhost:6379")
	}
	if os.Getenv("REDIS_PASSWORD") == "" {
		os.Setenv("REDIS_PASSWORD", "")
	}

	// Log test environment setup
	log.Println("🧪 Setting up Product Management test environment...")
	log.Printf("📊 Database: %s:%s/%s", os.Getenv("TEST_DB_HOST"), os.Getenv("TEST_DB_PORT"), os.Getenv("TEST_DB_NAME"))
	log.Printf("🔐 JWT Secret: %s", os.Getenv("JWT_SECRET"))
	log.Printf("📦 Redis: %s", os.Getenv("REDIS_ADDR"))

	// Run all tests
	exitCode := m.Run()

	// Cleanup and exit
	log.Println("🧹 Cleaning up test environment...")
	log.Printf("🏁 Tests completed with exit code: %d", exitCode)
	os.Exit(exitCode)
}
