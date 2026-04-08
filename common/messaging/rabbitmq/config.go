package rabbitmq

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Config holds RabbitMQ-specific connection settings.
type Config struct {
	URL      string
	Username string
	Password string
	Host     string
	Port     string
	VHost    string
	TLS      bool
}

// LoadConfigFromEnv loads RabbitMQ config from environment.
// URL takes precedence; otherwise DSN is composed from individual fields.
func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		URL:      strings.TrimSpace(os.Getenv("RABBITMQ_URL")),
		Username: strings.TrimSpace(os.Getenv("RABBITMQ_USER")),
		Password: strings.TrimSpace(os.Getenv("RABBITMQ_PASSWORD")),
		Host:     strings.TrimSpace(os.Getenv("RABBITMQ_HOST")),
		Port:     strings.TrimSpace(os.Getenv("RABBITMQ_PORT")),
		VHost:    strings.TrimSpace(os.Getenv("RABBITMQ_VHOST")),
		TLS:      strings.EqualFold(strings.TrimSpace(os.Getenv("RABBITMQ_TLS")), "true"),
	}

	if cfg.URL != "" {
		return cfg, nil
	}

	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == "" {
		cfg.Port = "5672"
	}
	if cfg.Username == "" || cfg.Password == "" {
		return Config{}, errors.New("rabbitmq credentials are required: set RABBITMQ_URL or RABBITMQ_USER/RABBITMQ_PASSWORD")
	}

	scheme := "amqp"
	if cfg.TLS {
		scheme = "amqps"
	}

	vhost := cfg.VHost
	if vhost == "" {
		vhost = "/"
	}

	cfg.URL = fmt.Sprintf(
		"%s://%s:%s@%s:%s/%s",
		scheme,
		url.QueryEscape(cfg.Username),
		url.QueryEscape(cfg.Password),
		cfg.Host,
		cfg.Port,
		strings.TrimPrefix(vhost, "/"),
	)

	return cfg, nil
}
