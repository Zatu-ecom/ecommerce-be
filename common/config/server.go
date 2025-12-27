package config

import (
	"fmt"
	"os"
)

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port string
	Mode string // "debug", "release", "test"
}

// loadServerConfig loads server configuration from environment variables.
func loadServerConfig() ServerConfig {
	return ServerConfig{
		Port: getEnvOrDefault("PORT", "8080"),
		Mode: getEnvOrDefault("GIN_MODE", "release"),
	}
}

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// Addr returns the server address in ":port" format.
func (s *ServerConfig) Addr() string {
	return fmt.Sprintf(":%s", s.Port)
}

// IsProduction returns true if running in release mode.
func (s *ServerConfig) IsProduction() bool {
	return s.Mode == "release"
}

// IsDebug returns true if running in debug mode.
func (s *ServerConfig) IsDebug() bool {
	return s.Mode == "debug"
}
