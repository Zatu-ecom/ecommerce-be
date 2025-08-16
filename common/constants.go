package common

import "time"

// Common constants for auth and redis functionality
const (
	// Redis constants
	RedisNotInitializedMsg = "redis client is not initialized"

	// Auth token constants
	TokenInvalidMsg = "invalid token"

	// Authentication messages
	AuthenticationRequiredMsg = "Authentication required"
	TokenRevokedMsg           = "Token has been revoked"
	InvalidAuthFormatMsg      = "Invalid authorization format"
	NoTokenProvidedMsg        = "No token provided"

	// Auth error codes
	AuthRequiredCode      = "AUTH_REQUIRED"
	TokenInvalidCode      = "TOKEN_INVALID"
	TokenRevokedCode      = "TOKEN_REVOKED"
	InvalidAuthFormatCode = "INVALID_AUTH_FORMAT"
	TokenRequiredCode     = "TOKEN_REQUIRED"

	// Context keys
	UserIDKey = "user_id"
	EmailKey  = "email"

	// Bearer token constants
	BearerPrefix = "Bearer"

	// Token expiration
	TokenExpirationTime = "24h"
)

// Time constants
const (
	TokenExpireDuration = time.Hour * 24
)
