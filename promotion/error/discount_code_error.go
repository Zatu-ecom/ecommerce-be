package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
)

// Discount code / coupon error codes (no magic strings).
const (
	DISCOUNT_CODE_NOT_FOUND_CODE           = "DISCOUNT_CODE_NOT_FOUND"
	DISCOUNT_CODE_EXISTS_CODE              = "DISCOUNT_CODE_EXISTS"
	UNAUTHORIZED_DISCOUNT_CODE_ACCESS_CODE = "UNAUTHORIZED_DISCOUNT_CODE_ACCESS"
	INVALID_DISCOUNT_CODE_VALUE_CODE       = "INVALID_DISCOUNT_CODE_VALUE"
	INVALID_DISCOUNT_CODE_DATE_RANGE_CODE  = "INVALID_DISCOUNT_CODE_DATE_RANGE"
	DISCOUNT_CODE_HAS_USAGE_CODE           = "DISCOUNT_CODE_HAS_USAGE"
	INVALID_DISCOUNT_CODE_SCOPE_CODE       = "INVALID_DISCOUNT_CODE_SCOPE"

	INVALID_COUPON_CODE              = "INVALID_COUPON"
	COUPON_EXPIRED_CODE              = "COUPON_EXPIRED"
	COUPON_NOT_STARTED_CODE          = "COUPON_NOT_STARTED"
	COUPON_USAGE_LIMIT_REACHED_CODE  = "COUPON_USAGE_LIMIT_REACHED"
	COUPON_ALREADY_USED_CODE         = "COUPON_ALREADY_USED"
	COUPON_MIN_PURCHASE_NOT_MET_CODE = "COUPON_MIN_PURCHASE_NOT_MET"
	COUPON_MIN_QUANTITY_NOT_MET_CODE = "COUPON_MIN_QUANTITY_NOT_MET"
	COUPON_NOT_ELIGIBLE_CODE         = "COUPON_NOT_ELIGIBLE"
	COUPON_CANNOT_COMBINE_CODE       = "COUPON_CANNOT_COMBINE"
	COUPON_ALREADY_APPLIED_CODE      = "COUPON_ALREADY_APPLIED"
	COUPON_NOT_APPLICABLE_CODE       = "COUPON_NOT_APPLICABLE"
	COUPON_NOT_ON_CART_CODE          = "COUPON_NOT_ON_CART"
	GUEST_COUPON_NOT_ALLOWED_CODE    = "GUEST_COUPON_NOT_ALLOWED"
)

// Discount code / coupon error messages.
const (
	DISCOUNT_CODE_NOT_FOUND_MSG           = "Discount code not found"
	DISCOUNT_CODE_EXISTS_MSG              = "Discount code already exists for this seller"
	UNAUTHORIZED_DISCOUNT_CODE_ACCESS_MSG = "You do not have permission to access this discount code"
	INVALID_DISCOUNT_CODE_VALUE_MSG       = "Invalid discount code value for the selected type"
	INVALID_DISCOUNT_CODE_DATE_RANGE_MSG  = "Invalid discount code date range"
	DISCOUNT_CODE_HAS_USAGE_MSG           = "Cannot delete discount code that has been used; deactivate it instead"
	INVALID_DISCOUNT_CODE_SCOPE_MSG       = "Scope mutation is not allowed for this discount code appliesTo value"

	INVALID_COUPON_MSG              = "Invalid or inactive coupon code"
	COUPON_EXPIRED_MSG              = "This coupon has expired"
	COUPON_NOT_STARTED_MSG          = "This coupon is not yet active"
	COUPON_USAGE_LIMIT_REACHED_MSG  = "This coupon has reached its usage limit"
	COUPON_ALREADY_USED_MSG         = "You have already used this coupon"
	COUPON_MIN_PURCHASE_NOT_MET_MSG = "Minimum purchase amount not met for this coupon"
	COUPON_MIN_QUANTITY_NOT_MET_MSG = "Minimum quantity not met for this coupon"
	COUPON_NOT_ELIGIBLE_MSG         = "You are not eligible for this coupon"
	COUPON_CANNOT_COMBINE_MSG       = "This coupon cannot be combined with other discounts"
	COUPON_ALREADY_APPLIED_MSG      = "This coupon is already applied to your cart"
	COUPON_NOT_APPLICABLE_MSG       = "Coupon is not applicable to items in your cart"
	COUPON_NOT_ON_CART_MSG          = "Coupon is not applied to your cart"
	GUEST_COUPON_NOT_ALLOWED_MSG    = "Coupons require an authenticated customer cart"
)

var (
	ErrDiscountCodeNotFound = &commonError.AppError{
		Code:       DISCOUNT_CODE_NOT_FOUND_CODE,
		Message:    DISCOUNT_CODE_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	ErrDiscountCodeExists = &commonError.AppError{
		Code:       DISCOUNT_CODE_EXISTS_CODE,
		Message:    DISCOUNT_CODE_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	ErrUnauthorizedDiscountCodeAccess = &commonError.AppError{
		Code:       UNAUTHORIZED_DISCOUNT_CODE_ACCESS_CODE,
		Message:    UNAUTHORIZED_DISCOUNT_CODE_ACCESS_MSG,
		StatusCode: http.StatusForbidden,
	}

	ErrInvalidDiscountCodeValue = &commonError.AppError{
		Code:       INVALID_DISCOUNT_CODE_VALUE_CODE,
		Message:    INVALID_DISCOUNT_CODE_VALUE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrInvalidDiscountCodeDateRange = &commonError.AppError{
		Code:       INVALID_DISCOUNT_CODE_DATE_RANGE_CODE,
		Message:    INVALID_DISCOUNT_CODE_DATE_RANGE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrDiscountCodeHasUsage = &commonError.AppError{
		Code:       DISCOUNT_CODE_HAS_USAGE_CODE,
		Message:    DISCOUNT_CODE_HAS_USAGE_MSG,
		StatusCode: http.StatusConflict,
	}

	ErrInvalidDiscountCodeScope = &commonError.AppError{
		Code:       INVALID_DISCOUNT_CODE_SCOPE_CODE,
		Message:    INVALID_DISCOUNT_CODE_SCOPE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrInvalidCoupon = &commonError.AppError{
		Code:       INVALID_COUPON_CODE,
		Message:    INVALID_COUPON_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCouponExpired = &commonError.AppError{
		Code:       COUPON_EXPIRED_CODE,
		Message:    COUPON_EXPIRED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCouponNotStarted = &commonError.AppError{
		Code:       COUPON_NOT_STARTED_CODE,
		Message:    COUPON_NOT_STARTED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCouponUsageLimitReached = &commonError.AppError{
		Code:       COUPON_USAGE_LIMIT_REACHED_CODE,
		Message:    COUPON_USAGE_LIMIT_REACHED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCouponAlreadyUsed = &commonError.AppError{
		Code:       COUPON_ALREADY_USED_CODE,
		Message:    COUPON_ALREADY_USED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCouponMinPurchaseNotMet = &commonError.AppError{
		Code:       COUPON_MIN_PURCHASE_NOT_MET_CODE,
		Message:    COUPON_MIN_PURCHASE_NOT_MET_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCouponMinQuantityNotMet = &commonError.AppError{
		Code:       COUPON_MIN_QUANTITY_NOT_MET_CODE,
		Message:    COUPON_MIN_QUANTITY_NOT_MET_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCouponNotEligible = &commonError.AppError{
		Code:       COUPON_NOT_ELIGIBLE_CODE,
		Message:    COUPON_NOT_ELIGIBLE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCouponCannotCombine = &commonError.AppError{
		Code:       COUPON_CANNOT_COMBINE_CODE,
		Message:    COUPON_CANNOT_COMBINE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCouponAlreadyApplied = &commonError.AppError{
		Code:       COUPON_ALREADY_APPLIED_CODE,
		Message:    COUPON_ALREADY_APPLIED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCouponNotApplicable = &commonError.AppError{
		Code:       COUPON_NOT_APPLICABLE_CODE,
		Message:    COUPON_NOT_APPLICABLE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCouponNotOnCart = &commonError.AppError{
		Code:       COUPON_NOT_ON_CART_CODE,
		Message:    COUPON_NOT_ON_CART_MSG,
		StatusCode: http.StatusNotFound,
	}

	ErrGuestCouponNotAllowed = &commonError.AppError{
		Code:       GUEST_COUPON_NOT_ALLOWED_CODE,
		Message:    GUEST_COUPON_NOT_ALLOWED_MSG,
		StatusCode: http.StatusForbidden,
	}
)
