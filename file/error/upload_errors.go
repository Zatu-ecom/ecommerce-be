package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/file/utils/constant"
)

// Upload-related AppError singletons.
//
// These correspond 1-to-1 with the error codes in research R11 and
// file/utils/constant/upload_constants.go.
//
// Usage in service layer:
//
//	return nil, fileError.ErrFileUploadForbidden
//
// Usage with custom message:
//
//	return nil, fileError.ErrFileUploadInvalidInput.WithMessage("filename must not be empty")
var (
	// ErrFileUploadUnauthorized (401) — missing or invalid authentication token.
	ErrFileUploadUnauthorized = commonError.NewAppError(
		constant.FILE_UPLOAD_UNAUTHORIZED_CODE,
		constant.FILE_UPLOAD_UNAUTHORIZED_MSG,
		http.StatusUnauthorized,
	)

	// ErrFileUploadForbidden (403) — authenticated but wrong role (e.g. customer).
	ErrFileUploadForbidden = commonError.NewAppError(
		constant.FILE_UPLOAD_FORBIDDEN_CODE,
		constant.FILE_UPLOAD_FORBIDDEN_MSG,
		http.StatusForbidden,
	)

	// ErrFileUploadInvalidInput (400) — request-body validation failure.
	ErrFileUploadInvalidInput = commonError.NewAppError(
		constant.FILE_UPLOAD_INVALID_INPUT_CODE,
		constant.FILE_UPLOAD_INVALID_INPUT_MSG,
		http.StatusBadRequest,
	)

	// ErrFileUploadPolicyViolation (422) — purpose/mime/size policy rejected.
	ErrFileUploadPolicyViolation = commonError.NewAppError(
		constant.FILE_UPLOAD_POLICY_VIOLATION_CODE,
		constant.FILE_UPLOAD_POLICY_VIOLATION_MSG,
		http.StatusUnprocessableEntity,
	)

	// ErrFileUploadStorageUnavailable (503) — blob adapter or presign call failed.
	ErrFileUploadStorageUnavailable = commonError.NewAppError(
		constant.FILE_UPLOAD_STORAGE_UNAVAILABLE_CODE,
		constant.FILE_UPLOAD_STORAGE_UNAVAILABLE_MSG,
		http.StatusServiceUnavailable,
	)

	// ErrFileUploadNoStorageConfig (412) — no active, validated storage config exists.
	ErrFileUploadNoStorageConfig = commonError.NewAppError(
		constant.FILE_UPLOAD_NO_STORAGE_CONFIG_CODE,
		constant.FILE_UPLOAD_NO_STORAGE_CONFIG_MSG,
		http.StatusPreconditionFailed,
	)

	// ErrFileUploadNotFound (404) — fileId not found OR cross-tenant access attempt.
	// Always 404 — never 403 — to prevent enumeration attacks.
	ErrFileUploadNotFound = commonError.NewAppError(
		constant.FILE_UPLOAD_NOT_FOUND_CODE,
		constant.FILE_UPLOAD_NOT_FOUND_MSG,
		http.StatusNotFound,
	)

	// ErrFileUploadConflict (409) — Idempotency-Key fingerprint mismatch or the
	// row has already moved past UPLOADING before a retry.
	ErrFileUploadConflict = commonError.NewAppError(
		constant.FILE_UPLOAD_CONFLICT_CODE,
		constant.FILE_UPLOAD_CONFLICT_MSG,
		http.StatusConflict,
	)

	// ErrFileUploadObjectMissing (409) — complete-upload called before PUT to provider.
	ErrFileUploadObjectMissing = commonError.NewAppError(
		constant.FILE_UPLOAD_OBJECT_MISSING_CODE,
		constant.FILE_UPLOAD_OBJECT_MISSING_MSG,
		http.StatusConflict,
	)

	// ErrFileUploadObjectMismatch (422) — ETag, size, or content-type verification at
	// complete time failed. Row is transitioned to FAILED/OBJECT_MISMATCH.
	ErrFileUploadObjectMismatch = commonError.NewAppError(
		constant.FILE_UPLOAD_OBJECT_MISMATCH_CODE,
		constant.FILE_UPLOAD_OBJECT_MISMATCH_MSG,
		http.StatusUnprocessableEntity,
	)

	// ErrFileUploadExpired (410) — complete-upload called after the upload window expired
	// and the scheduler handler already transitioned the row to FAILED/UPLOAD_EXPIRED.
	ErrFileUploadExpired = commonError.NewAppError(
		constant.FILE_UPLOAD_EXPIRED_CODE,
		constant.FILE_UPLOAD_EXPIRED_MSG,
		http.StatusGone,
	)

	// ErrFileUploadInternal (500) — catch-all for unexpected/unhandled errors.
	// Message is always stripped of provider details before returning to the caller.
	ErrFileUploadInternal = commonError.NewAppError(
		constant.FILE_UPLOAD_INTERNAL_CODE,
		constant.FILE_UPLOAD_INTERNAL_MSG,
		http.StatusInternalServerError,
	)
)
