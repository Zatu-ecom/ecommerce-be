package config

import (
	"os"
	"time"
)

// AuthConfig holds JWT and authentication configuration.
type AuthConfig struct {
	JWTSecret      string
	JWTExpiryHours int
}

// loadAuthConfig loads auth configuration from environment variables.
func loadAuthConfig() AuthConfig {
	return AuthConfig{
		JWTSecret:      os.Getenv("JWT_SECRET"),
		JWTExpiryHours: getEnvAsIntOrDefault("JWT_EXPIRY_HOURS", 24),
	}
}

// TokenExpiry returns the JWT token expiry duration.
func (a *AuthConfig) TokenExpiry() time.Duration {
	return time.Duration(a.JWTExpiryHours) * time.Hour
}
