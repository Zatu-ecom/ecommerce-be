package db

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var db *gorm.DB

func ConnectDB(autoMigrations []AutoMigrate) {
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

	fmt.Println(dsn)

	/* Initialize database */
	_db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // Use singular table names
		},
	})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	db = _db
	fmt.Println("🚀 Database connected successfully!")

	// /* Auto-migrate tables */
	for _, migration := range autoMigrations {
		err := db.AutoMigrate(migration.AutoMigrate()...)
		if err != nil {
			log.Fatal("Failed to migrate database: ", err)
		}
	}
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
		log.Printf("Error getting SQL DB instance: %v", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		log.Printf("Error closing database connection: %v", err)
	} else {
		log.Println("Database connection closed")
	}
}

type AutoMigrate interface {
	AutoMigrate() []interface{}
}
