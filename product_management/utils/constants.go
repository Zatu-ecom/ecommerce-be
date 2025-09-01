package utils

import (
	"ecommerce-be/common"
)

// Import auth-related constants from common package
const (
	// Use auth constants from common package
	AuthenticationRequiredMsg = common.AuthenticationRequiredMsg
	TokenInvalidMsg           = common.TokenInvalidMsg
	TokenRevokedMsg           = common.TokenRevokedMsg
	InvalidAuthFormatMsg      = common.InvalidAuthFormatMsg
	NoTokenProvidedMsg        = common.NoTokenProvidedMsg

	// Auth error codes
	AuthRequiredCode      = common.AuthRequiredCode
	TokenInvalidCode      = common.TokenInvalidCode
	TokenRevokedCode      = common.TokenRevokedCode
	InvalidAuthFormatCode = common.InvalidAuthFormatCode
	TokenRequiredCode     = common.TokenRequiredCode

	// Context keys
	UserIDKey = common.UserIDKey
	EmailKey  = common.EmailKey

	// Token settings
	TokenExpireDuration = common.TokenExpireDuration

	// Redis constants (use from common)
	RedisNotInitializedMsg = common.RedisNotInitializedMsg
)
