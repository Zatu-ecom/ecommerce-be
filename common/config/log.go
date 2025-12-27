package config

import (
	"os"
	"strings"
)

// LogConfig holds logging configuration.
type LogConfig struct {
	Level           string
	ExtendedLogging bool
}

// loadLogConfig loads logging configuration from environment variables.
func loadLogConfig() LogConfig {
	return LogConfig{
		Level:           getEnvOrDefault("LOG_LEVEL", "info"),
		ExtendedLogging: strings.ToLower(os.Getenv("EXTENDED_LOGGING")) == "true",
	}
}

// IsDebug returns true if log level is debug.
func (l *LogConfig) IsDebug() bool {
	return l.Level == "debug"
}
