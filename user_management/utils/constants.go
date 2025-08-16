package utils

import (
	"datun.com/be/common"
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

	// User error messages
	UserExistsMsg             = "User with this email already exists"
	UserNotFoundMsg           = "User not found"
	InvalidCredentialsMsg     = "Invalid email or password"
	AccountDeactivatedMsg     = "Account is deactivated"
	PasswordMismatchMsg       = "New password and confirmation do not match"
	InvalidCurrentPasswordMsg = "Current password is incorrect"
	InvalidRequestFormatMsg   = "Invalid request format"
	ValidationFailedMsg       = "Validation failed"

	// Operation failure messages
	FailedToRegisterUserMsg      = "Failed to register user"
	FailedToRefreshTokenMsg      = "Failed to refresh token"
	FailedToGetProfileMsg        = "Failed to get profile"
	FailedToUpdateProfileMsg     = "Failed to update profile"
	FailedToChangePasswordMsg    = "Failed to change password"
	FailedToGetAddressesMsg      = "Failed to get addresses"
	FailedToAddAddressMsg        = "Failed to add address"
	FailedToUpdateAddressMsg     = "Failed to update address"
	FailedToDeleteAddressMsg     = "Failed to delete address"
	FailedToSetDefaultAddressMsg = "Failed to set default address"

	// Permission and access messages
	PermissionDeniedMsg    = "You don't have permission to update this address"
	CannotDeleteDefaultMsg = "Cannot delete default address. Please set another address as default first."
	InvalidAddressIDMsg    = "Invalid address ID"

	// Request field names
	RequestFieldName = "request"

	// Response field names
	UserFieldName      = "user"
	TokenFieldName     = "token"
	ExpiresInFieldName = "expiresIn"
	AddressFieldName   = "address"
	AddressesFieldName = "addresses"

	// Time constants
	TokenExpirationDisplay = "24h"

	// User error codes
	UserExistsCode             = "USER_EXISTS"
	UserNotFoundCode           = "USER_NOT_FOUND"
	InvalidCredentialsCode     = "INVALID_CREDENTIALS"
	AccountDeactivatedCode     = "ACCOUNT_DEACTIVATED"
	ValidationErrorCode        = "VALIDATION_ERROR"
	PasswordMismatchCode       = "PASSWORD_MISMATCH"
	InvalidCurrentPasswordCode = "INVALID_CURRENT_PASSWORD"
	PermissionDeniedCode       = "PERMISSION_DENIED"
	InvalidIDCode              = "INVALID_ID"
	CannotDeleteDefaultCode    = "CANNOT_DELETE_DEFAULT"

	// Address error messages
	AddressNotFoundMsg                = "Address not found"
	DefaultAddressExistsMsg           = "Default address already exists"
	CannotDeleteOnlyDefaultAddressMsg = "cannot delete the only default address"

	// Address error codes
	AddressNotFoundCode      = "ADDRESS_NOT_FOUND"
	DefaultAddressExistsCode = "DEFAULT_ADDRESS_EXISTS"

	// Response messages
	SuccessMsg               = "Success"
	RegisterSuccessMsg       = "User registered successfully"
	LoginSuccessMsg          = "Login successful"
	LogoutSuccessMsg         = "Logged out successfully"
	ProfileRetrievedMsg      = "Profile retrieved successfully"
	ProfileUpdatedMsg        = "Profile updated successfully"
	PasswordChangedMsg       = "Password changed successfully"
	TokenRefreshedMsg        = "Token refreshed successfully"
	AddressCreatedMsg        = "Address created successfully"
	AddressUpdatedMsg        = "Address updated successfully"
	AddressDeletedMsg        = "Address deleted successfully"
	AddressesRetrievedMsg    = "Addresses retrieved successfully"
	DefaultAddressUpdatedMsg = "Default address updated successfully"
)
