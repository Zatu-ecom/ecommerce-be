package config

import "sync"

// Config holds all application configuration grouped by concern.
// This is the main struct that embeds all sub-configs.
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Auth      AuthConfig
	App       AppConfig
	Log       LogConfig
	Scheduler SchedulerConfig
}

var (
	instance *Config
	once     sync.Once
)

// Get returns the singleton Config instance.
// Must call Load() first to initialize.
func Get() *Config {
	return instance
}

// Reset clears the singleton instance (for testing purposes).
func Reset() {
	instance = nil
	once = sync.Once{}
}
