package db

import (
	"fmt"
	"os"

	"ecommerce-be/common/log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var db *gorm.DB

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

	/* Initialize database */
	_db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // Use singular table names
		},
	})
	if err != nil {
		log.Fatal("Failed to connect to database", err)
	}

	db = _db
	log.Info("Database connected successfully")
}

func GetDB() *gorm.DB {
	return db
}

func SetDB(database *gorm.DB) {
	db = database
}

func Atomic(fn func(tx *gorm.DB) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// CloseDB closes the database connection
// TODO: We are not use this function anywhere. we can remove this function
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
