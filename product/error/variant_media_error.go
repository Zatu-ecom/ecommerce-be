package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/utils"
)

// Variant Media Errors

var (
	// ErrVariantMediaNotFound is returned when the requested variant-media link does not exist.
	ErrVariantMediaNotFound = &commonError.AppError{
		Code:       utils.VARIANT_MEDIA_NOT_FOUND_CODE,
		Message:    utils.VARIANT_MEDIA_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrVariantMediaDuplicate is returned when the same file is attached to a variant more than once.
	ErrVariantMediaDuplicate = &commonError.AppError{
		Code:       utils.VARIANT_MEDIA_DUPLICATE_CODE,
		Message:    utils.VARIANT_MEDIA_DUPLICATE_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrVariantMediaInvalidFile is returned when the referenced file does not exist or is inaccessible.
	ErrVariantMediaInvalidFile = &commonError.AppError{
		Code:       utils.VARIANT_MEDIA_INVALID_FILE_CODE,
		Message:    utils.VARIANT_MEDIA_INVALID_FILE_MSG,
		StatusCode: http.StatusUnprocessableEntity,
	}
)
