package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/file/utils/constant"
)

var (
	ErrFileNotFound = commonError.NewAppError(
		constant.FILE_NOT_FOUND_CODE,
		constant.FILE_NOT_FOUND_MSG,
		http.StatusNotFound,
	)

	ErrFileNotActive = commonError.NewAppError(
		constant.FILE_NOT_ACTIVE_CODE,
		constant.FILE_NOT_ACTIVE_MSG,
		http.StatusConflict,
	)

	ErrVariantNotFound = commonError.NewAppError(
		constant.VARIANT_NOT_FOUND_CODE,
		constant.VARIANT_NOT_FOUND_MSG,
		http.StatusNotFound,
	)

	ErrVariantNotReady = commonError.NewAppError(
		constant.VARIANT_NOT_READY_CODE,
		constant.VARIANT_NOT_READY_MSG,
		http.StatusConflict,
	)

	ErrFileDeleteConflict = commonError.NewAppError(
		constant.FILE_DELETE_CONFLICT_CODE,
		constant.FILE_DELETE_CONFLICT_MSG,
		http.StatusConflict,
	)

	ErrStoragePermissionDenied = commonError.NewAppError(
		constant.STORAGE_PERMISSION_DENIED_CODE,
		constant.STORAGE_PERMISSION_DENIED_MSG,
		http.StatusBadGateway,
	)

	ErrStorageUnavailable = commonError.NewAppError(
		constant.STORAGE_UNAVAILABLE_CODE,
		constant.STORAGE_UNAVAILABLE_MSG,
		http.StatusServiceUnavailable,
	)
)
