package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
)

const (
	SALE_NOT_FOUND_CODE               = "SALE_NOT_FOUND"
	SALE_SLUG_EXISTS_CODE             = "SALE_SLUG_EXISTS"
	UNAUTHORIZED_SALE_ACCESS_CODE     = "UNAUTHORIZED_SALE_ACCESS"
	INVALID_SALE_DATE_RANGE_CODE      = "INVALID_SALE_DATE_RANGE"
	INVALID_SALE_FOR_PROMOTION_CODE   = "INVALID_SALE_FOR_PROMOTION"
	SALE_INVALID_FILE_CODE            = "SALE_INVALID_FILE"
)

const (
	SALE_NOT_FOUND_MSG               = "Sale not found"
	SALE_SLUG_EXISTS_MSG             = "Sale with this slug already exists for this seller"
	UNAUTHORIZED_SALE_ACCESS_MSG     = "You do not have permission to access this sale"
	INVALID_SALE_DATE_RANGE_MSG      = "Invalid sale date range"
	INVALID_SALE_FOR_PROMOTION_MSG   = "Sale is invalid or does not belong to this seller"
	SALE_INVALID_FILE_MSG            = "One or more banner files are invalid or not accessible"
)

var (
	ErrSaleNotFound = &commonError.AppError{
		Code:       SALE_NOT_FOUND_CODE,
		Message:    SALE_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	ErrSaleSlugExists = &commonError.AppError{
		Code:       SALE_SLUG_EXISTS_CODE,
		Message:    SALE_SLUG_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	ErrUnauthorizedSaleAccess = &commonError.AppError{
		Code:       UNAUTHORIZED_SALE_ACCESS_CODE,
		Message:    UNAUTHORIZED_SALE_ACCESS_MSG,
		StatusCode: http.StatusForbidden,
	}

	ErrInvalidSaleDateRange = &commonError.AppError{
		Code:       INVALID_SALE_DATE_RANGE_CODE,
		Message:    INVALID_SALE_DATE_RANGE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrInvalidSaleForPromotion = &commonError.AppError{
		Code:       INVALID_SALE_FOR_PROMOTION_CODE,
		Message:    INVALID_SALE_FOR_PROMOTION_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrSaleInvalidFile = &commonError.AppError{
		Code:       SALE_INVALID_FILE_CODE,
		Message:    SALE_INVALID_FILE_MSG,
		StatusCode: http.StatusUnprocessableEntity,
	}
)
