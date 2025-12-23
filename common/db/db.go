package db

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"ecommerce-be/common/log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var db *gorm.DB

// ConnectDB initializes a PostgreSQL database connection using GORM.
// It configures the connection pool, sets up logging based on APP_ENV,
// and verifies the connection with a ping. Panics if connection fails.
//
// Required environment variables:
//   - DB_HOST, DB_USER, DB_PASSWORD, DB_NAME, DB_PORT, DB_SSLMODE
//
// Optional environment variables (with defaults):
//   - DB_MAX_OPEN_CONNS (25) - Maximum open connections
//   - DB_MAX_IDLE_CONNS (10) - Maximum idle connections
//   - DB_CONN_MAX_LIFETIME_MINUTES (30) - Connection max lifetime
//   - DB_CONN_MAX_IDLE_TIME_MINUTES (5) - Connection max idle time
//   - APP_ENV ("prod" for error-only logging)
func ConnectDB() {
	/* PostgreSQL connection string */
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	log.Info(
		"Connecting to database host: " + os.Getenv(
			"DB_HOST",
		) + ", dbname: " + os.Getenv(
			"DB_NAME",
		) + ", port: " + os.Getenv(
			"DB_PORT",
		),
	)

	/* Configure logger based on environment */
	logLevel := logger.Info
	if os.Getenv("APP_ENV") == "prod" {
		logLevel = logger.Error
	}

	/* Initialize database */
	_db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // Use singular table names
		},
	})
	if err != nil {
		log.Fatal("Failed to connect to database", err)
	}

	/* Configure connection pool for production */
	configureConnectionPool(_db)

	db = _db
	log.Info("Database connected successfully")
}

func configureConnectionPool(_db *gorm.DB) {
	sqlDB, err := _db.DB()
	if err != nil {
		log.Fatal("Failed to get underlying SQL DB", err)
	}

	// Connection pool settings (can be overridden via env vars)
	maxOpenConns := getEnvAsInt("DB_MAX_OPEN_CONNS", 25)
	maxIdleConns := getEnvAsInt("DB_MAX_IDLE_CONNS", 10)
	connMaxLifetimeMinutes := getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)
	connMaxIdleTimeMinutes := getEnvAsInt("DB_CONN_MAX_IDLE_TIME_MINUTES", 5)

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetimeMinutes) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(connMaxIdleTimeMinutes) * time.Minute)

	/* Verify connection is actually working */
	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Failed to ping database", err)
	}
}

// getEnvAsInt reads an environment variable as int with a default fallback
func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func GetDB() *gorm.DB {
	return db
}

func SetDB(database *gorm.DB) {
	db = database
}

// CloseDB closes the database connection
func CloseDB() {
	sqlDB, err := db.DB()
	if err != nil {
		log.Error("Error getting SQL DB instance", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		log.Error("Error closing database connection", err)
	} else {
		log.Info("Database connection closed")
	}
}
