package error

import (
	"fmt"
	"net/http"

	commonError "ecommerce-be/common/error"
)

// ErrVariantNotFound is returned when variant API fails to find a variant
var ErrVariantNotFound = &commonError.AppError{
	Code:       "VARIANT_NOT_FOUND",
	Message:    "Unable to fetch pricing for a variant in cart.",
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
