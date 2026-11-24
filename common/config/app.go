package config

// AppConfig holds general application configuration.
type AppConfig struct {
	Env string // "dev", "staging", "prod"

	// Wishlist limits
	MaxWishlistsPerUser int
	MaxWishlistItems    int

	// Recently viewed queue size (max products tracked per user)
	RecentlyViewedQueueSize int

	// System encryption key (32 bytes ideal for AES-256)
	EncryptionKey string

	// PublicAPIBaseURL is the externally reachable base URL used to compose
	// provider-facing callback URLs (e.g. webhookUrl).
	// Env: PUBLIC_API_BASE_URL (default http://localhost:8080).
	PublicAPIBaseURL string

	// PaymentPendingTTLMinutes bounds how long a pending transaction may stay
	// open before the reconciler treats it as expired unpaid.
	// Env: PAYMENT_PENDING_TTL_MINUTES (default 45).
	PaymentPendingTTLMinutes int

	// PaymentSessionlessTTLMinutes bounds how long a pending transaction without
	// a gateway session id may stay open before the reconciler fails it.
	// Env: PAYMENT_SESSIONLESS_TTL_MINUTES (default 5).
	PaymentSessionlessTTLMinutes int

	// RefundStuckTTLMinutes bounds how long a pending/processing refund may stay
	// unresolved before the reconciler re-checks it with the provider.
	// Env: REFUND_STUCK_TTL_MINUTES (default 30).
	RefundStuckTTLMinutes int
}

// loadAppConfig loads app configuration from environment variables.
func loadAppConfig() AppConfig {
	return AppConfig{
		Env:                     getEnvOrDefault("APP_ENV", "dev"),
		MaxWishlistsPerUser:     getEnvAsIntOrDefault("MAX_WISHLISTS_PER_USER", 10),
		MaxWishlistItems:        getEnvAsIntOrDefault("MAX_WISHLIST_ITEMS", 100),
		RecentlyViewedQueueSize: getEnvAsIntOrDefault("RECENTLY_VIEWED_QUEUE_SIZE", 10),
		EncryptionKey: getEnvOrDefault(
			"ENCRYPTION_KEY",
			"0123456789abcdef0123456789abcdef",
		), // Default 32-byte key for local dev
		PublicAPIBaseURL:             getEnvOrDefault("PUBLIC_API_BASE_URL", "http://localhost:8080"),
		PaymentPendingTTLMinutes:     getEnvAsIntOrDefault("PAYMENT_PENDING_TTL_MINUTES", 45),
		PaymentSessionlessTTLMinutes: getEnvAsIntOrDefault("PAYMENT_SESSIONLESS_TTL_MINUTES", 5),
		RefundStuckTTLMinutes:        getEnvAsIntOrDefault("REFUND_STUCK_TTL_MINUTES", 30),
	}
}

// IsProduction returns true if running in production environment.
func (a *AppConfig) IsProduction() bool {
	return a.Env == "prod"
}

// IsDevelopment returns true if running in development environment.
func (a *AppConfig) IsDevelopment() bool {
	return a.Env == "dev" || a.Env == ""
}

func (a *AppConfig) IsLocal() bool {
	return a.Env == "local"
}
