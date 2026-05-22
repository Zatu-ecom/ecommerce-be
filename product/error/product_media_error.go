package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/utils"
)

// Product Media Errors

var (
	// ErrProductMediaNotFound is returned when the requested product-media link does not exist.
	ErrProductMediaNotFound = &commonError.AppError{
		Code:       utils.PRODUCT_MEDIA_NOT_FOUND_CODE,
		Message:    utils.PRODUCT_MEDIA_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrProductMediaDuplicate is returned when the same file is attached to a product more than once.
	ErrProductMediaDuplicate = &commonError.AppError{
		Code:       utils.PRODUCT_MEDIA_DUPLICATE_CODE,
		Message:    utils.PRODUCT_MEDIA_DUPLICATE_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrProductMediaInvalidFile is returned when the referenced file does not exist or is inaccessible.
	ErrProductMediaInvalidFile = &commonError.AppError{
		Code:       utils.PRODUCT_MEDIA_INVALID_FILE_CODE,
		Message:    utils.PRODUCT_MEDIA_INVALID_FILE_MSG,
		StatusCode: http.StatusUnprocessableEntity,
	}

	// ErrProductMediaCleanupFailed is a degradation sentinel used for logging only.
	// The product-media removal itself is still reported as successful to the caller.
	ErrProductMediaCleanupFailed = &commonError.AppError{
		Code:       utils.PRODUCT_MEDIA_CLEANUP_FAILED_CODE,
		Message:    utils.PRODUCT_MEDIA_CLEANUP_FAILED_MSG,
		StatusCode: http.StatusInternalServerError,
	}
)
