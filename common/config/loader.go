package config

import (
	"errors"
	"os"
	"strconv"
)

// Load reads environment variables and initializes the singleton Config.
// Returns an error if required configuration is missing.
func Load() (*Config, error) {
	var loadErr error

	once.Do(func() {
		cfg := &Config{
			Server:    loadServerConfig(),
			Database:  loadDatabaseConfig(),
			Redis:     loadRedisConfig(),
			Auth:      loadAuthConfig(),
			App:       loadAppConfig(),
			Log:       loadLogConfig(),
			Scheduler: loadSchedulerConfig(),
		}

		if err := cfg.Validate(); err != nil {
			loadErr = err
			return
		}

		instance = cfg
	})

	if loadErr != nil {
		return nil, loadErr
	}

	return instance, nil
}

// Validate checks that all required configuration is present.
func (c *Config) Validate() error {
	// Database validation
	if c.Database.Host == "" {
		return errors.New("DB_HOST is required")
	}
	if c.Database.User == "" {
		return errors.New("DB_USER is required")
	}
	if c.Database.Name == "" {
		return errors.New("DB_NAME is required")
	}
	if c.Database.Port == "" {
		return errors.New("DB_PORT is required")
	}

	// Redis validation
	if c.Redis.Host == "" {
		return errors.New("REDIS_ADDR is required")
	}

	// Auth validation
	if c.Auth.JWTSecret == "" {
		return errors.New("JWT_SECRET is required")
	}

	return nil
}

// getEnvAsIntOrDefault reads an environment variable as int with a default fallback.
func getEnvAsIntOrDefault(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
