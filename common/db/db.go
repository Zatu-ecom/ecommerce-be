package db

import (
	"time"

	"ecommerce-be/common/config"
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
func ConnectDB(cfg *config.Config) {
	dsn := cfg.Database.DSN()

	log.Info("Connecting to database: " + cfg.Database.LogSafeString())

	/* Configure logger based on environment */
	logLevel := logger.Info
	if cfg.App.IsProduction() {
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
	configureConnectionPool(_db, cfg)

	db = _db
	log.Info("Database connected successfully")
}

func configureConnectionPool(_db *gorm.DB, cfg *config.Config) {
	sqlDB, err := _db.DB()
	if err != nil {
		log.Fatal("Failed to get underlying SQL DB", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetimeMinutes) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.Database.ConnMaxIdleTimeMinutes) * time.Minute)

	/* Verify connection is actually working */
	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Failed to ping database", err)
	}
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
