package config

import (
	"fmt"
	"os"
)

// DatabaseConfig holds PostgreSQL database configuration.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string

	// Connection pool settings
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeMinutes int
	ConnMaxIdleTimeMinutes int
}

// loadDatabaseConfig loads database configuration from environment variables.
func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:                   os.Getenv("DB_HOST"),
		Port:                   os.Getenv("DB_PORT"),
		User:                   os.Getenv("DB_USER"),
		Password:               os.Getenv("DB_PASSWORD"),
		Name:                   os.Getenv("DB_NAME"),
		SSLMode:                getEnvOrDefault("DB_SSLMODE", "disable"),
		MaxOpenConns:           getEnvAsIntOrDefault("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:           getEnvAsIntOrDefault("DB_MAX_IDLE_CONNS", 10),
		ConnMaxLifetimeMinutes: getEnvAsIntOrDefault("DB_CONN_MAX_LIFETIME_MINUTES", 30),
		ConnMaxIdleTimeMinutes: getEnvAsIntOrDefault("DB_CONN_MAX_IDLE_TIME_MINUTES", 5),
	}
}

// DSN returns the PostgreSQL connection string.
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		d.Host,
		d.User,
		d.Password,
		d.Name,
		d.Port,
		d.SSLMode,
	)
}

// LogSafeString returns a connection string safe for logging (no password).
func (d *DatabaseConfig) LogSafeString() string {
	return fmt.Sprintf("host=%s dbname=%s port=%s", d.Host, d.Name, d.Port)
}
