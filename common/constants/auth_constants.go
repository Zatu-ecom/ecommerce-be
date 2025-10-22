package constants

import "time"

// Authentication-related constants
const (
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
	USER_ID_KEY    = "user_id"
	EMAIL_KEY      = "email"
	ROLE_ID_KEY    = "role_id"
	ROLE_NAME_KEY  = "role_name"
	ROLE_LEVEL_KEY = "role_level"
	SELLER_ID_KEY  = "seller_id"

	// Header keys
	SELLER_ID_HEADER = "X-Seller-ID"

	// Seller validation messages
	SELLER_ID_REQUIRED_MSG = "Seller ID is required in X-Seller-ID header"
	SELLER_ID_INVALID_MSG  = "Invalid seller ID provided"

	// Seller validation error codes
	SELLER_ID_REQUIRED_CODE = "SELLER_ID_REQUIRED"
	SELLER_ID_INVALID_CODE  = "SELLER_ID_INVALID"

	// Bearer token constants
	BEARER_PREFIX = "Bearer"

	// Token expiration
	TOKEN_EXPIRATION_TIME = "24h"
)

// Time constants
const (
	TOKEN_EXPIRE_DURATION = time.Hour * 24
)
