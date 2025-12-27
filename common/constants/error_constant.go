package constants

const (
	ROLE_DATA_MISSING_CODE      = "ROLE_DATA_MISSING"
	UNAUTHORIZED_ERROR_CODE     = "UNAUTHORIZED"
	VALIDATION_ERROR_CODE       = "VALIDATION_ERROR"
	NO_FIELDS_PROVIDED_CODE     = "NO_FIELDS_PROVIDED"
	INVALID_REQUEST_STRUCT_CODE = "INVALID_REQUEST_STRUCT"
	SELLER_DATA_MISSING_CODE    = "SELLER_DATA_MISSING"
	REQUIRED_QUERY_PARAM_CODE   = "REQUIRED_QUERY_PARAM_MISSING"
	INVALID_LIMIT_CODE          = "INVALID_LIMIT"
	USER_DATA_MISSING_CODE      = "USER_DATA_MISSING"
	CORRELATION_ID_MISSING      = "CORRELATION_ID_MISSING"
)

const (
	ROLE_DATA_MISSING_MSG      = "Role data is missing in the context"
	UNAUTHORIZED_ERROR_MSG     = "Unauthorized access"
	VALIDATION_FAILED_MSG      = "Validation failed"
	INVALID_REQUEST_FORMAT_MSG = "Invalid request format"
	NO_FIELDS_PROVIDED_MSG     = "At least one field must be provided for update"
	INVALID_REQUEST_STRUCT_MSG = "Invalid request structure"
	SELLER_DATA_MISSING_MSG    = "Seller data is missing"
	REQUIRED_QUERY_PARAM_MSG   = "Required query parameter is missing"
	INVALID_LIMIT_MSG          = "Limit must be between 1 and 100"
	USER_DATA_MISSING_MSG      = "User data is missing in the context"
	CORRELATION_ID_MISSING_MSG = "Correlation ID is missing in the context"
)

const (
	REQUEST_FIELD_NAME = "request"
)
