package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/file/utils/constant"
)

// Blob adapter sentinel errors.
// Each operation that encounters a provider error wraps the appropriate
// sentinel using .WithMessagef() to add operation-specific context.
var (
	// ErrBlobNotFound is returned when the requested object, key, or bucket
	// does not exist in the storage provider.
	ErrBlobNotFound = commonError.NewAppError(
		constant.BLOB_ADAPTER_NOT_FOUND_CODE,
		constant.BLOB_ADAPTER_NOT_FOUND_MSG,
		http.StatusNotFound,
	)

	// ErrBlobPermissionDenied is returned when the storage provider rejects
	// the request due to invalid credentials or insufficient permissions.
	// Raw credentials are never included in the error message.
	ErrBlobPermissionDenied = commonError.NewAppError(
		constant.BLOB_ADAPTER_PERMISSION_DENIED_CODE,
		constant.BLOB_ADAPTER_PERMISSION_DENIED_MSG,
		http.StatusForbidden,
	)

	// ErrBlobNetwork is returned for timeouts, connection failures, and DNS
	// errors when communicating with the storage provider.
	ErrBlobNetwork = commonError.NewAppError(
		constant.BLOB_ADAPTER_NETWORK_ERR_CODE,
		constant.BLOB_ADAPTER_NETWORK_ERR_MSG,
		http.StatusServiceUnavailable,
	)

	// ErrBlobValidation is returned when the caller supplies invalid parameters
	// (e.g. zero or negative TTL, missing required fields) before any SDK call.
	ErrBlobValidation = commonError.NewAppError(
		constant.BLOB_ADAPTER_VALIDATION_ERR_CODE,
		constant.BLOB_ADAPTER_VALIDATION_ERR_MSG,
		http.StatusBadRequest,
	)

	// ErrBlobInternal is returned for unexpected SDK or runtime failures that
	// do not match a more specific category.
	ErrBlobInternal = commonError.NewAppError(
		constant.BLOB_ADAPTER_INTERNAL_ERR_CODE,
		constant.BLOB_ADAPTER_INTERNAL_ERR_MSG,
		http.StatusInternalServerError,
	)

	// ErrBlobFactoryInit is returned when the adapter factory fails to construct
	// a provider adapter (decryption failure, missing fields, unknown type).
	ErrBlobFactoryInit = commonError.NewAppError(
		constant.BLOB_FACTORY_INIT_ERR_CODE,
		constant.BLOB_FACTORY_INIT_ERR_MSG,
		http.StatusInternalServerError,
	)
)

// IsBlobError reports whether err is an *AppError whose Code matches the given sentinel.
// Use this instead of errors.Is because WithMessagef returns a new pointer, not the sentinel.
func IsBlobError(err error, sentinel *commonError.AppError) bool {
	appErr, ok := commonError.AsAppError(err)
	if !ok {
		return false
	}
	return appErr.Code == sentinel.Code
}
