package utils

import (
	"ecommerce-be/common"
)

// Import auth-related constants from common package
const (
	// Use auth constants from common package
	AUTHENTICATION_REQUIRED_MSG = common.AUTHENTICATION_REQUIRED_MSG
	TOKEN_INVALID_MSG           = common.TOKEN_INVALID_MSG
	TOKEN_REVOKED_MSG           = common.TOKEN_REVOKED_MSG
	INVALID_AUTH_FORMAT_MSG     = common.INVALID_AUTH_FORMAT_MSG
	NO_TOKEN_PROVIDED_MSG       = common.NO_TOKEN_PROVIDED_MSG

	// Auth error codes
	AUTH_REQUIRED_CODE       = common.AUTH_REQUIRED_CODE
	TOKEN_INVALID_CODE       = common.TOKEN_INVALID_CODE
	TOKEN_REVOKED_CODE       = common.TOKEN_REVOKED_CODE
	INVALID_AUTH_FORMAT_CODE = common.INVALID_AUTH_FORMAT_CODE
	TOKEN_REQUIRED_CODE      = common.TOKEN_REQUIRED_CODE

	// Context keys
	USER_ID_KEY = common.USER_ID_KEY
	EMAIL_KEY   = common.EMAIL_KEY

	// Token settings
	TOKEN_EXPIRE_DURATION = common.TOKEN_EXPIRE_DURATION

	// Redis constants (use from common)
	REDIS_NOT_INITIALIZED_MSG = common.REDIS_NOT_INITIALIZED_MSG
)

// Backward-compatible aliases (to be removed after migration)
const (
	AuthenticationRequiredMsg = AUTHENTICATION_REQUIRED_MSG
	TokenInvalidMsg           = TOKEN_INVALID_MSG
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

	TokenExpireDuration = TOKEN_EXPIRE_DURATION

	RedisNotInitializedMsg = REDIS_NOT_INITIALIZED_MSG
)
