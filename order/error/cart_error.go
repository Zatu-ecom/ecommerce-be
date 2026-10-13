package error

import (
	"fmt"
	"net/http"

	commonError "ecommerce-be/common/error"
)

// ErrVariantNotFound is returned when variant API fails to find a variant
var ErrVariantNotFound = &commonError.AppError{
	Code:       "VARIANT_NOT_FOUND",
	Message:    "Unable to fetch variant information",
	StatusCode: http.StatusBadRequest,
}

// ErrInsufficientStock returns an error for insufficient stock
func ErrInsufficientStock(available int) *commonError.AppError {
	return &commonError.AppError{
		Code:       "INSUFFICIENT_STOCK",
		Message:    fmt.Sprintf("Only %d items available for this variant", available),
		StatusCode: http.StatusBadRequest,
	}
}

// ErrPromotionServiceUnavailable returns an error when promotion service fails
func ErrPromotionServiceUnavailable(err error) *commonError.AppError {
	return &commonError.AppError{
		Code:       "SYSTEM_ERROR",
		Message:    "Promotion service unavailable: " + err.Error(),
		StatusCode: http.StatusInternalServerError,
	}
}

// ErrDeviceCartNotFound is returned when no active cart exists for the given device ID
var ErrDeviceCartNotFound = &commonError.AppError{
	Code:       "DEVICE_CART_NOT_FOUND",
	Message:    "No cart found for this device",
	StatusCode: http.StatusNotFound,
}

// ErrCartMergeFailed is returned when merging device cart into user cart encounters an unexpected error
func ErrCartMergeFailed(err error) *commonError.AppError {
	return &commonError.AppError{
		Code:       "CART_MERGE_FAILED",
		Message:    "Failed to merge guest cart: " + err.Error(),
		StatusCode: http.StatusInternalServerError,
	}
}

// ErrCouponApplyRateLimited is returned when a customer applies coupons too frequently
var ErrCouponApplyRateLimited = &commonError.AppError{
	Code:       "COUPON_APPLY_RATE_LIMITED",
	Message:    "Too many coupon apply attempts. Please try again shortly",
	StatusCode: http.StatusTooManyRequests,
}
