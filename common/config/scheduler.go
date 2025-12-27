package config

// SchedulerConfig holds background job scheduler configuration.
type SchedulerConfig struct {
	WorkerPoolSize int
}

// loadSchedulerConfig loads scheduler configuration from environment variables.
func loadSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		WorkerPoolSize: getEnvAsIntOrDefault("WORKER_POOL_SIZE", 5),
	}
}
