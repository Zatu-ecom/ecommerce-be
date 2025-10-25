package setup

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// getAbsolutePath returns the absolute path for a relative path from the project root
func getAbsolutePath(relativePath string) (string, error) {
	_, b, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get caller information")
	}
	basepath := filepath.Dir(b)
	return filepath.Join(basepath, "../../..", relativePath), nil
}

// runSQLFile executes a SQL file
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
