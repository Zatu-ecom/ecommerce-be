package config

import "time"

// PasswordResetConfig holds password reset token configuration.
type PasswordResetConfig struct {
	TokenExpiryHours int // How many hours before a reset token expires
	TokenByteLength  int // Length of the random token in bytes (32 = 256 bits)
}

// loadPasswordResetConfig loads password reset configuration from environment variables.
func loadPasswordResetConfig() PasswordResetConfig {
	return PasswordResetConfig{
		TokenExpiryHours: getEnvAsIntOrDefault("RESET_TOKEN_EXPIRY_HOURS", 1),
		TokenByteLength:  getEnvAsIntOrDefault("RESET_TOKEN_BYTE_LENGTH", 32),
	}
}

// TokenExpiry returns the reset token expiry duration.
func (p *PasswordResetConfig) TokenExpiry() time.Duration {
	return time.Duration(p.TokenExpiryHours) * time.Hour
}
