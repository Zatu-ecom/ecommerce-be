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

	// User error messages
	USER_EXISTS_MSG              = "User with this email already exists"
	USER_NOT_FOUND_MSG           = "User not found"
	INVALID_CREDENTIALS_MSG      = "Invalid email or password"
	ACCOUNT_DEACTIVATED_MSG      = "Account is deactivated"
	PASSWORD_MISMATCH_MSG        = "New password and confirmation do not match"
	INVALID_CURRENT_PASSWORD_MSG = "Current password is incorrect"
	INVALID_REQUEST_FORMAT_MSG   = "Invalid request format"
	VALIDATION_FAILED_MSG        = "Validation failed"

	// Operation failure messages
	FAILED_TO_REGISTER_USER_MSG       = "Failed to register user"
	FAILED_TO_REFRESH_TOKEN_MSG       = "Failed to refresh token"
	FAILED_TO_GET_PROFILE_MSG         = "Failed to get profile"
	FAILED_TO_UPDATE_PROFILE_MSG      = "Failed to update profile"
	FAILED_TO_CHANGE_PASSWORD_MSG     = "Failed to change password"
	FAILED_TO_GET_ADDRESSES_MSG       = "Failed to get addresses"
	FAILED_TO_ADD_ADDRESS_MSG         = "Failed to add address"
	FAILED_TO_UPDATE_ADDRESS_MSG      = "Failed to update address"
	FAILED_TO_DELETE_ADDRESS_MSG      = "Failed to delete address"
	FAILED_TO_SET_DEFAULT_ADDRESS_MSG = "Failed to set default address"

	// Permission and access messages
	PERMISSION_DENIED_MSG     = "You don't have permission to update this address"
	CANNOT_DELETE_DEFAULT_MSG = "Cannot delete default address. Please set another address as default first."
	INVALID_ADDRESS_ID_MSG    = "Invalid address ID"

	// Request field names
	REQUEST_FIELD_NAME = "request"

	// Response field names
	USER_FIELD_NAME       = "user"
	TOKEN_FIELD_NAME      = "token"
	EXPIRES_IN_FIELD_NAME = "expiresIn"
	ADDRESS_FIELD_NAME    = "address"
	ADDRESSES_FIELD_NAME  = "addresses"

	// Time constants
	TOKEN_EXPIRATION_DISPLAY = "24h"

	// User error codes
	USER_EXISTS_CODE              = "USER_EXISTS"
	USER_NOT_FOUND_CODE           = "USER_NOT_FOUND"
	INVALID_CREDENTIALS_CODE      = "INVALID_CREDENTIALS"
	ACCOUNT_DEACTIVATED_CODE      = "ACCOUNT_DEACTIVATED"
	VALIDATION_ERROR_CODE         = "VALIDATION_ERROR"
	PASSWORD_MISMATCH_CODE        = "PASSWORD_MISMATCH"
	INVALID_CURRENT_PASSWORD_CODE = "INVALID_CURRENT_PASSWORD"
	PERMISSION_DENIED_CODE        = "PERMISSION_DENIED"
	INVALID_ID_CODE               = "INVALID_ID"
	CANNOT_DELETE_DEFAULT_CODE    = "CANNOT_DELETE_DEFAULT"

	// Address error messages
	ADDRESS_NOT_FOUND_MSG                  = "Address not found"
	DEFAULT_ADDRESS_EXISTS_MSG             = "Default address already exists"
	CANNOT_DELETE_ONLY_DEFAULT_ADDRESS_MSG = "cannot delete the only default address"

	// Address error codes
	ADDRESS_NOT_FOUND_CODE      = "ADDRESS_NOT_FOUND"
	DEFAULT_ADDRESS_EXISTS_CODE = "DEFAULT_ADDRESS_EXISTS"

	// Response messages
	SUCCESS_MSG                 = "Success"
	REGISTER_SUCCESS_MSG        = "User registered successfully"
	LOGIN_SUCCESS_MSG           = "Login successful"
	LOGOUT_SUCCESS_MSG          = "Logged out successfully"
	PROFILE_RETRIEVED_MSG       = "Profile retrieved successfully"
	PROFILE_UPDATED_MSG         = "Profile updated successfully"
	PASSWORD_CHANGED_MSG        = "Password changed successfully"
	TOKEN_REFRESHED_MSG         = "Token refreshed successfully"
	ADDRESS_CREATED_MSG         = "Address created successfully"
	ADDRESS_UPDATED_MSG         = "Address updated successfully"
	ADDRESS_DELETED_MSG         = "Address deleted successfully"
	ADDRESSES_RETRIEVED_MSG     = "Addresses retrieved successfully"
	DEFAULT_ADDRESS_UPDATED_MSG = "Default address updated successfully"
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

	TokenExpireDuration    = TOKEN_EXPIRE_DURATION
	TokenExpirationDisplay = TOKEN_EXPIRATION_DISPLAY

	UserExistsMsg             = USER_EXISTS_MSG
	UserNotFoundMsg           = USER_NOT_FOUND_MSG
	InvalidCredentialsMsg     = INVALID_CREDENTIALS_MSG
	AccountDeactivatedMsg     = ACCOUNT_DEACTIVATED_MSG
	PasswordMismatchMsg       = PASSWORD_MISMATCH_MSG
	InvalidCurrentPasswordMsg = INVALID_CURRENT_PASSWORD_MSG
	InvalidRequestFormatMsg   = INVALID_REQUEST_FORMAT_MSG
	ValidationFailedMsg       = VALIDATION_FAILED_MSG

	FailedToRegisterUserMsg      = FAILED_TO_REGISTER_USER_MSG
	FailedToRefreshTokenMsg      = FAILED_TO_REFRESH_TOKEN_MSG
	FailedToGetProfileMsg        = FAILED_TO_GET_PROFILE_MSG
	FailedToUpdateProfileMsg     = FAILED_TO_UPDATE_PROFILE_MSG
	FailedToChangePasswordMsg    = FAILED_TO_CHANGE_PASSWORD_MSG
	FailedToGetAddressesMsg      = FAILED_TO_GET_ADDRESSES_MSG
	FailedToAddAddressMsg        = FAILED_TO_ADD_ADDRESS_MSG
	FailedToUpdateAddressMsg     = FAILED_TO_UPDATE_ADDRESS_MSG
	FailedToDeleteAddressMsg     = FAILED_TO_DELETE_ADDRESS_MSG
	FailedToSetDefaultAddressMsg = FAILED_TO_SET_DEFAULT_ADDRESS_MSG

	PermissionDeniedMsg    = PERMISSION_DENIED_MSG
	CannotDeleteDefaultMsg = CANNOT_DELETE_DEFAULT_MSG
	InvalidAddressIDMsg    = INVALID_ADDRESS_ID_MSG

	RequestFieldName   = REQUEST_FIELD_NAME
	UserFieldName      = USER_FIELD_NAME
	TokenFieldName     = TOKEN_FIELD_NAME
	ExpiresInFieldName = EXPIRES_IN_FIELD_NAME
	AddressFieldName   = ADDRESS_FIELD_NAME
	AddressesFieldName = ADDRESSES_FIELD_NAME

	UserExistsCode             = USER_EXISTS_CODE
	UserNotFoundCode           = USER_NOT_FOUND_CODE
	InvalidCredentialsCode     = INVALID_CREDENTIALS_CODE
	AccountDeactivatedCode     = ACCOUNT_DEACTIVATED_CODE
	ValidationErrorCode        = VALIDATION_ERROR_CODE
	PasswordMismatchCode       = PASSWORD_MISMATCH_CODE
	InvalidCurrentPasswordCode = INVALID_CURRENT_PASSWORD_CODE
	PermissionDeniedCode       = PERMISSION_DENIED_CODE
	InvalidIDCode              = INVALID_ID_CODE
	CannotDeleteDefaultCode    = CANNOT_DELETE_DEFAULT_CODE

	AddressNotFoundMsg                = ADDRESS_NOT_FOUND_MSG
	DefaultAddressExistsMsg           = DEFAULT_ADDRESS_EXISTS_MSG
	CannotDeleteOnlyDefaultAddressMsg = CANNOT_DELETE_ONLY_DEFAULT_ADDRESS_MSG

	AddressNotFoundCode      = ADDRESS_NOT_FOUND_CODE
	DefaultAddressExistsCode = DEFAULT_ADDRESS_EXISTS_CODE

	SuccessMsg               = SUCCESS_MSG
	RegisterSuccessMsg       = REGISTER_SUCCESS_MSG
	LoginSuccessMsg          = LOGIN_SUCCESS_MSG
	LogoutSuccessMsg         = LOGOUT_SUCCESS_MSG
	ProfileRetrievedMsg      = PROFILE_RETRIEVED_MSG
	ProfileUpdatedMsg        = PROFILE_UPDATED_MSG
	PasswordChangedMsg       = PASSWORD_CHANGED_MSG
	TokenRefreshedMsg        = TOKEN_REFRESHED_MSG
	AddressCreatedMsg        = ADDRESS_CREATED_MSG
	AddressUpdatedMsg        = ADDRESS_UPDATED_MSG
	AddressDeletedMsg        = ADDRESS_DELETED_MSG
	AddressesRetrievedMsg    = ADDRESSES_RETRIEVED_MSG
	DefaultAddressUpdatedMsg = DEFAULT_ADDRESS_UPDATED_MSG
)
