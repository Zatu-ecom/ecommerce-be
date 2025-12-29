package constant

// ========================================
// USER ERROR CODES
// ========================================
const (
	USER_EXISTS_CODE              = "USER_EXISTS"
	USER_NOT_FOUND_CODE           = "USER_NOT_FOUND"
	INVALID_CREDENTIALS_CODE      = "INVALID_CREDENTIALS"
	ACCOUNT_DEACTIVATED_CODE      = "ACCOUNT_DEACTIVATED"
	VALIDATION_ERROR_CODE         = "VALIDATION_ERROR"
	PASSWORD_MISMATCH_CODE        = "PASSWORD_MISMATCH"
	INVALID_CURRENT_PASSWORD_CODE = "INVALID_CURRENT_PASSWORD"
	PERMISSION_DENIED_CODE        = "PERMISSION_DENIED"
	INVALID_ID_CODE               = "INVALID_ID"
)

// ========================================
// USER ERROR MESSAGES
// ========================================
const (
	USER_EXISTS_MSG              = "User with this email already exists"
	USER_NOT_FOUND_MSG           = "User not found"
	INVALID_CREDENTIALS_MSG      = "Invalid email or password"
	ACCOUNT_DEACTIVATED_MSG      = "Account is deactivated"
	PASSWORD_MISMATCH_MSG        = "New password and confirmation do not match"
	INVALID_CURRENT_PASSWORD_MSG = "Current password is incorrect"
	INVALID_REQUEST_FORMAT_MSG   = "Invalid request format"
	VALIDATION_FAILED_MSG        = "Validation failed"
)

// ========================================
// USER OPERATION FAILURE MESSAGES
// ========================================
const (
	FAILED_TO_REGISTER_USER_MSG   = "Failed to register user"
	FAILED_TO_REFRESH_TOKEN_MSG   = "Failed to refresh token"
	FAILED_TO_GET_PROFILE_MSG     = "Failed to get profile"
	FAILED_TO_UPDATE_PROFILE_MSG  = "Failed to update profile"
	FAILED_TO_CHANGE_PASSWORD_MSG = "Failed to change password"
	FAILED_TO_LIST_USERS_MSG      = "Failed to list users"
)

// ========================================
// USER SUCCESS MESSAGES
// ========================================
const (
	SUCCESS_MSG            = "Success"
	REGISTER_SUCCESS_MSG   = "User registered successfully"
	LOGIN_SUCCESS_MSG      = "Login successful"
	LOGOUT_SUCCESS_MSG     = "Logged out successfully"
	PROFILE_RETRIEVED_MSG  = "Profile retrieved successfully"
	PROFILE_UPDATED_MSG    = "Profile updated successfully"
	PASSWORD_CHANGED_MSG   = "Password changed successfully"
	TOKEN_REFRESHED_MSG    = "Token refreshed successfully"
	USERS_RETRIEVED_MSG    = "Users retrieved successfully"
)

// ========================================
// USER PERMISSION MESSAGES
// ========================================
const (
	PERMISSION_DENIED_MSG      = "You don't have permission to update this address"
	USER_LIST_UNAUTHORIZED_MSG = "You don't have permission to list users"
)

// ========================================
// USER FIELD NAMES
// ========================================
const (
	REQUEST_FIELD_NAME    = "request"
	USER_FIELD_NAME       = "user"
	TOKEN_FIELD_NAME      = "token"
	EXPIRES_IN_FIELD_NAME = "expiresIn"
)
