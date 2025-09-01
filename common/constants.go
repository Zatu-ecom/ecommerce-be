package common

import "time"

// Common constants for auth and redis functionality
const (
	// Redis constants
	REDIS_NOT_INITIALIZED_MSG = "redis client is not initialized"

	// Auth token constants
	TOKEN_INVALID_MSG = "invalid token"

	// Authentication messages
	AUTHENTICATION_REQUIRED_MSG = "Authentication required"
	TOKEN_REVOKED_MSG           = "Token has been revoked"
	INVALID_AUTH_FORMAT_MSG     = "Invalid authorization format"
	NO_TOKEN_PROVIDED_MSG       = "No token provided"

	// Auth error codes
	AUTH_REQUIRED_CODE       = "AUTH_REQUIRED"
	TOKEN_INVALID_CODE       = "TOKEN_INVALID"
	TOKEN_REVOKED_CODE       = "TOKEN_REVOKED"
	INVALID_AUTH_FORMAT_CODE = "INVALID_AUTH_FORMAT"
	TOKEN_REQUIRED_CODE      = "TOKEN_REQUIRED"

	// Context keys
	USER_ID_KEY = "user_id"
	EMAIL_KEY   = "email"

	// Bearer token constants
	BEARER_PREFIX = "Bearer"

	// Token expiration
	TOKEN_EXPIRATION_TIME = "24h"
)

// Time constants
const (
	TOKEN_EXPIRE_DURATION = time.Hour * 24
)

// Backward-compatible aliases (to be removed after migration)
const (
	RedisNotInitializedMsg = REDIS_NOT_INITIALIZED_MSG
	TokenInvalidMsg        = TOKEN_INVALID_MSG

	AuthenticationRequiredMsg = AUTHENTICATION_REQUIRED_MSG
	TokenRevokedMsg           = TOKEN_REVOKED_MSG
	InvalidAuthFormatMsg      = INVALID_AUTH_FORMAT_MSG
	NoTokenProvidedMsg        = NO_TOKEN_PROVIDED_MSG

	AuthRequiredCode      = AUTH_REQUIRED_CODE
	TokenInvalidCode      = TOKEN_INVALID_CODE
	TokenRevokedCode      = TOKEN_REVOKED_CODE
	InvalidAuthFormatCode = INVALID_AUTH_FORMAT_CODE
	TokenRequiredCode     = TOKEN_REQUIRED_CODE

	UserIDKey = USER_ID_KEY
	EmailKey  = EMAIL_KEY

	BearerPrefix        = BEARER_PREFIX
	TokenExpirationTime = TOKEN_EXPIRATION_TIME

	TokenExpireDuration = TOKEN_EXPIRE_DURATION
)
