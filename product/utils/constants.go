package utils

import (
	"ecommerce-be/common/constants"
)

// Import auth-related constants from common package
const (
	// Use auth constants from common package
	AUTHENTICATION_REQUIRED_MSG = constants.AUTHENTICATION_REQUIRED_MSG
	TOKEN_INVALID_MSG           = constants.TOKEN_INVALID_MSG
	TOKEN_REVOKED_MSG           = constants.TOKEN_REVOKED_MSG
	INVALID_AUTH_FORMAT_MSG     = constants.INVALID_AUTH_FORMAT_MSG
	NO_TOKEN_PROVIDED_MSG       = constants.NO_TOKEN_PROVIDED_MSG

	// Auth error codes
	AUTH_REQUIRED_CODE       = constants.AUTH_REQUIRED_CODE
	TOKEN_INVALID_CODE       = constants.TOKEN_INVALID_CODE
	TOKEN_REVOKED_CODE       = constants.TOKEN_REVOKED_CODE
	INVALID_AUTH_FORMAT_CODE = constants.INVALID_AUTH_FORMAT_CODE
	TOKEN_REQUIRED_CODE      = constants.TOKEN_REQUIRED_CODE

	// Context keys
	USER_ID_KEY = constants.USER_ID_KEY
	EMAIL_KEY   = constants.EMAIL_KEY

	// Token settings
	TOKEN_EXPIRE_DURATION = constants.TOKEN_EXPIRE_DURATION

	// Redis constants (use from common)
	REDIS_NOT_INITIALIZED_MSG = constants.REDIS_NOT_INITIALIZED_MSG
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
