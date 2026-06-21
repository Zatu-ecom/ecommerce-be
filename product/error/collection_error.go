package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/utils"
)

var (
	ErrCollectionNotFound = &commonError.AppError{
		Code:       utils.COLLECTION_NOT_FOUND_CODE,
		Message:    utils.COLLECTION_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	ErrCollectionExists = &commonError.AppError{
		Code:       utils.COLLECTION_EXISTS_CODE,
		Message:    utils.COLLECTION_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	ErrUnauthorizedCollectionAccess = &commonError.AppError{
		Code:       utils.UNAUTHORIZED_COLLECTION_ACCESS_CODE,
		Message:    utils.UNAUTHORIZED_COLLECTION_ACCESS_MSG,
		StatusCode: http.StatusForbidden,
	}

	ErrProductNotInCollection = &commonError.AppError{
		Code:       utils.PRODUCT_NOT_IN_COLLECTION_CODE,
		Message:    utils.PRODUCT_NOT_IN_COLLECTION_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrInvalidCollectionProduct = &commonError.AppError{
		Code:       utils.INVALID_COLLECTION_PRODUCT_CODE,
		Message:    utils.INVALID_COLLECTION_PRODUCT_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCollectionInvalidFile = &commonError.AppError{
		Code:       utils.COLLECTION_INVALID_FILE_CODE,
		Message:    utils.COLLECTION_INVALID_FILE_MSG,
		StatusCode: http.StatusUnprocessableEntity,
	}
)
