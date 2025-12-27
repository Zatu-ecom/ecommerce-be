package config

import "os"

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	Addr     string
}

// loadRedisConfig loads Redis configuration from environment variables.
func loadRedisConfig() RedisConfig {
	return RedisConfig{
		Host:     os.Getenv("REDIS_HOST"),
		Password: os.Getenv("REDIS_PASSWORD"),
		Port:     getEnvOrDefault("REDIS_PORT", "6379"),
		DB:       getEnvAsIntOrDefault("REDIS_DB", 0),
		Addr:     os.Getenv("REDIS_HOST") + ":" + getEnvOrDefault("REDIS_PORT", "6379"),
	}
}

// HasPassword returns true if a password is configured.
func (r *RedisConfig) HasPassword() bool {
	return r.Password != ""
}
