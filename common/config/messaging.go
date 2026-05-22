package config

import (
	"os"
	"strings"

	"ecommerce-be/common/messaging"
)

// MessagingConfig holds broker-level messaging configuration.
type MessagingConfig struct {
	Enabled bool
	// QueueType values: rabbitmq, kafka
	QueueType messaging.QueueType

	CommandsExchange string
	EventsExchange   string

	Prefetch            int
	ConsumerConcurrency int
	RetryDelayMS        int
	MaxRetries          int
}

// loadMessagingConfig loads messaging configuration from environment variables.
func loadMessagingConfig() MessagingConfig {
	rawEnabled := strings.ToLower(getEnvOrDefault("MESSAGING_ENABLED", "false")) == "true"
	queueType := messaging.ParseQueueType(getEnvOrDefault("MESSAGING_QUEUE_TYPE", "rabbitmq"))

	return MessagingConfig{
		Enabled:             rawEnabled,
		QueueType:           queueType,
		CommandsExchange:    firstNonEmptyEnv("MESSAGING_COMMANDS_EXCHANGE", "RABBITMQ_EXCHANGE_COMMANDS", "ecom.commands"),
		EventsExchange:      firstNonEmptyEnv("MESSAGING_EVENTS_EXCHANGE", "RABBITMQ_EXCHANGE_EVENTS", "ecom.events"),
		Prefetch:            getEnvAsIntOrDefault("MESSAGING_PREFETCH", 10),
		ConsumerConcurrency: getEnvAsIntOrDefault("MESSAGING_CONSUMER_CONCURRENCY", 5),
		RetryDelayMS:        getEnvAsIntOrDefault("MESSAGING_RETRY_DELAY_MS", 10000),
		MaxRetries:          getEnvAsIntOrDefault("MESSAGING_MAX_RETRIES", 5),
	}
}

func firstNonEmptyEnv(primary, legacy, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(primary)); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(legacy)); v != "" {
		return v
	}
	return fallback
}
