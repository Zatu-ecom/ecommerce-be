package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
)

// Promotion error codes
const (
	PROMOTION_NOT_FOUND_CODE            = "PROMOTION_NOT_FOUND"
	PROMOTION_SLUG_EXISTS_CODE          = "PROMOTION_SLUG_EXISTS"
	INVALID_DISCOUNT_CONFIG_CODE        = "INVALID_DISCOUNT_CONFIG"
	INVALID_DATE_RANGE_CODE             = "INVALID_DATE_RANGE"
	INVALID_ELIGIBILITY_CODE            = "INVALID_ELIGIBILITY"
	UNAUTHORIZED_PROMOTION_ACCESS_CODE  = "UNAUTHORIZED_PROMOTION_ACCESS"
	INVALID_STATUS_TRANSITION_CODE      = "INVALID_STATUS_TRANSITION"
	CANNOT_DELETE_ACTIVE_PROMOTION_CODE = "CANNOT_DELETE_ACTIVE_PROMOTION"
	CANNOT_EDIT_ACTIVE_PROMOTION_CODE   = "CANNOT_EDIT_ACTIVE_PROMOTION"
	CANNOT_EDIT_TERMINAL_PROMOTION_CODE = "CANNOT_EDIT_TERMINAL_PROMOTION"
	PROMOTION_UPDATE_FAILED_CODE        = "PROMOTION_UPDATE_FAILED"
	PROMOTION_DELETE_FAILED_CODE        = "PROMOTION_DELETE_FAILED"
)

// Promotion error messages
const (
	PROMOTION_NOT_FOUND_MSG            = "Promotion not found"
	PROMOTION_SLUG_EXISTS_MSG          = "Promotion with this slug already exists"
	INVALID_DISCOUNT_CONFIG_MSG        = "Invalid discount configuration"
	INVALID_DATE_RANGE_MSG             = "Invalid date range"
	INVALID_ELIGIBILITY_MSG            = "Invalid eligibility configuration"
	UNAUTHORIZED_PROMOTION_ACCESS_MSG  = "Unauthorized promotion access"
	INVALID_STATUS_TRANSITION_MSG      = "Invalid promotion status transition"
	CANNOT_DELETE_ACTIVE_PROMOTION_MSG = "Cannot delete an active promotion. Please deactivate it first."
	CANNOT_EDIT_ACTIVE_PROMOTION_MSG   = "Cannot edit critical fields of an active promotion. Pause or end it first."
	CANNOT_EDIT_TERMINAL_PROMOTION_MSG = "Cannot edit a promotion that has ended or been cancelled."
	PROMOTION_UPDATE_FAILED_MSG        = "Failed to update promotion"
	PROMOTION_DELETE_FAILED_MSG         = "Failed to delete promotion"
)

var (
	// ErrPromotionNotFound is returned when a promotion is not found
	ErrPromotionNotFound = &commonError.AppError{
		Code:       PROMOTION_NOT_FOUND_CODE,
		Message:    PROMOTION_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrPromotionSlugExists is returned when a promotion with the same slug already exists
	ErrPromotionSlugExists = &commonError.AppError{
		Code:       PROMOTION_SLUG_EXISTS_CODE,
		Message:    PROMOTION_SLUG_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrInvalidDiscountConfig is returned when discount config is invalid
	ErrInvalidDiscountConfig = &commonError.AppError{
		Code:       INVALID_DISCOUNT_CONFIG_CODE,
		Message:    INVALID_DISCOUNT_CONFIG_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrInvalidDateRange is returned when date range is invalid
	ErrInvalidDateRange = &commonError.AppError{
		Code:       INVALID_DATE_RANGE_CODE,
		Message:    INVALID_DATE_RANGE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrInvalidEligibility is returned when eligibility configuration is invalid
	ErrInvalidEligibility = &commonError.AppError{
		Code:       INVALID_ELIGIBILITY_CODE,
		Message:    INVALID_ELIGIBILITY_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrUnauthorizedPromotionAccess is returned when user doesn't have access to promotion
	ErrUnauthorizedPromotionAccess = &commonError.AppError{
		Code:       UNAUTHORIZED_PROMOTION_ACCESS_CODE,
		Message:    UNAUTHORIZED_PROMOTION_ACCESS_MSG,
		StatusCode: http.StatusForbidden,
	}

	// ErrInvalidStatusTransition is returned when a promotion status transition is invalid
	ErrInvalidStatusTransition = &commonError.AppError{
		Code:       INVALID_STATUS_TRANSITION_CODE,
		Message:    INVALID_STATUS_TRANSITION_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrCannotDeleteActivePromotion is returned when attempting to delete an active promotion
	ErrCannotDeleteActivePromotion = &commonError.AppError{
		Code:       CANNOT_DELETE_ACTIVE_PROMOTION_CODE,
		Message:    CANNOT_DELETE_ACTIVE_PROMOTION_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrCannotEditActivePromotion is returned when attempting to edit critical fields of an active promotion
	ErrCannotEditActivePromotion = &commonError.AppError{
		Code:       CANNOT_EDIT_ACTIVE_PROMOTION_CODE,
		Message:    CANNOT_EDIT_ACTIVE_PROMOTION_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrCannotEditTerminalPromotion is returned when attempting to edit a promotion that has ended or been cancelled
	ErrCannotEditTerminalPromotion = &commonError.AppError{
		Code:       CANNOT_EDIT_TERMINAL_PROMOTION_CODE,
		Message:    CANNOT_EDIT_TERMINAL_PROMOTION_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrPromotionUpdateFailed is returned when a promotion update fails
	ErrPromotionUpdateFailed = &commonError.AppError{
		Code:       PROMOTION_UPDATE_FAILED_CODE,
		Message:    PROMOTION_UPDATE_FAILED_MSG,
		StatusCode: http.StatusInternalServerError,
	}

	// ErrPromotionDeleteFailed is returned when a promotion delete fails
	ErrPromotionDeleteFailed = &commonError.AppError{
		Code:       PROMOTION_DELETE_FAILED_CODE,
		Message:    PROMOTION_DELETE_FAILED_MSG,
		StatusCode: http.StatusInternalServerError,
	}
)
