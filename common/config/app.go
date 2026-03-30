package config

// AppConfig holds general application configuration.
type AppConfig struct {
	Env string // "dev", "staging", "prod"

	// Wishlist limits
	MaxWishlistsPerUser int
	MaxWishlistItems    int
}

// loadAppConfig loads app configuration from environment variables.
func loadAppConfig() AppConfig {
	return AppConfig{
		Env:                 getEnvOrDefault("APP_ENV", "dev"),
		MaxWishlistsPerUser: getEnvAsIntOrDefault("MAX_WISHLISTS_PER_USER", 10),
		MaxWishlistItems:    getEnvAsIntOrDefault("MAX_WISHLIST_ITEMS", 100),
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
