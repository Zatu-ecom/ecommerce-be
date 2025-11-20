package setup

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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

// RunAllMigrations automatically discovers and runs all migration files in order
// Migrations are expected to be in the migrations/ directory and numbered (e.g., 001_*.sql, 002_*.sql)
func (tc *TestContainers) RunAllMigrations(t *testing.T) {
	migrationsDir := "migrations"
	absPath, err := getAbsolutePath(migrationsDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path for migrations directory: %v", err)
	}

	// Read all files in migrations directory
	files, err := os.ReadDir(absPath)
	if err != nil {
		t.Fatalf("Failed to read migrations directory %s: %v", absPath, err)
	}

	// Filter and sort migration files (exclude seeds directory and README)
	var migrationFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue // Skip directories like seeds/
		}
		fileName := file.Name()
		if !strings.HasSuffix(fileName, ".sql") {
			continue // Skip non-SQL files
		}
		if strings.Contains(strings.ToLower(fileName), "readme") {
			continue // Skip README files
		}
		migrationFiles = append(migrationFiles, fileName)
	}

	// Sort files to ensure they run in order (001, 002, 003, etc.)
	sort.Strings(migrationFiles)

	if len(migrationFiles) == 0 {
		// t.Logf("No migration files found in %s", absPath)
		return
	}

	t.Logf("Running %d migration(s) from %s", len(migrationFiles), migrationsDir)
	for _, fileName := range migrationFiles {
		migrationPath := filepath.Join(migrationsDir, fileName)
		t.Logf("  - Running migration: %s", fileName)
		tc.RunMigrations(t, migrationPath)
	}
	// t.Logf("All migrations completed successfully")
}

// RunAllSeeds automatically discovers and runs all seed files in order
// Seeds are expected to be in the migrations/seeds/ directory and numbered (e.g., 001_*.sql, 002_*.sql)
func (tc *TestContainers) RunAllSeeds(t *testing.T) {
	seedsDir := "migrations/seeds"
	absPath, err := getAbsolutePath(seedsDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path for seeds directory: %v", err)
	}

	// Read all files in seeds directory
	files, err := os.ReadDir(absPath)
	if err != nil {
		t.Fatalf("Failed to read seeds directory %s: %v", absPath, err)
	}

	// Filter and sort seed files
	var seedFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileName := file.Name()
		if !strings.HasSuffix(fileName, ".sql") {
			continue // Skip non-SQL files
		}
		seedFiles = append(seedFiles, fileName)
	}

	// Sort files to ensure they run in order
	sort.Strings(seedFiles)

	if len(seedFiles) == 0 {
		// t.Logf("No seed files found in %s", absPath)
		return
	}

	t.Logf("Running %d seed(s) from %s", len(seedFiles), seedsDir)
	for _, fileName := range seedFiles {
		seedPath := filepath.Join(seedsDir, fileName)
		t.Logf("  - Running seed: %s", fileName)
		tc.RunSeeds(t, seedPath)
	}
	// t.Logf("All seeds completed successfully")
}
