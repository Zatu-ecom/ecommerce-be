package constant

import (
	"ecommerce-be/common/constants"
)

// ========================================
// AUTH CONSTANTS (imported from common)
// ========================================

// Auth error messages
const (
	AUTHENTICATION_REQUIRED_MSG = constants.AUTHENTICATION_REQUIRED_MSG
	TOKEN_INVALID_MSG           = constants.TOKEN_INVALID_MSG
	TOKEN_REVOKED_MSG           = constants.TOKEN_REVOKED_MSG
	INVALID_AUTH_FORMAT_MSG     = constants.INVALID_AUTH_FORMAT_MSG
	NO_TOKEN_PROVIDED_MSG       = constants.NO_TOKEN_PROVIDED_MSG
)

// Auth error codes
const (
	AUTH_REQUIRED_CODE       = constants.AUTH_REQUIRED_CODE
	TOKEN_INVALID_CODE       = constants.TOKEN_INVALID_CODE
	TOKEN_REVOKED_CODE       = constants.TOKEN_REVOKED_CODE
	INVALID_AUTH_FORMAT_CODE = constants.INVALID_AUTH_FORMAT_CODE
	TOKEN_REQUIRED_CODE      = constants.TOKEN_REQUIRED_CODE
)

// Context keys
const (
	USER_ID_KEY = constants.USER_ID_KEY
	EMAIL_KEY   = constants.EMAIL_KEY
)

// Token settings
const (
	TOKEN_EXPIRE_DURATION = constants.TOKEN_EXPIRE_DURATION
)

// Redis constants
const (
	REDIS_NOT_INITIALIZED_MSG = constants.REDIS_NOT_INITIALIZED_MSG
)
