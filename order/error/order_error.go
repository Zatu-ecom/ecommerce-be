package error

import (
	"fmt"
	"net/http"

	commonError "ecommerce-be/common/error"
)

const (
	ORDER_CART_NOT_ACTIVE_CODE         = "ORDER_CART_NOT_ACTIVE"
	ORDER_CART_EMPTY_CODE              = "ORDER_CART_EMPTY"
	ORDER_CART_ALREADY_CHECKOUT_CODE   = "ORDER_CART_ALREADY_IN_CHECKOUT"
	ORDER_NOT_FOUND_CODE               = "ORDER_NOT_FOUND"
	ORDER_INVALID_STATUS_CODE          = "ORDER_INVALID_STATUS"
	ORDER_INVALID_TRANSITION_CODE      = "ORDER_INVALID_STATUS_TRANSITION"
	ORDER_TRANSACTION_ID_REQUIRED_CODE = "ORDER_TRANSACTION_ID_REQUIRED"
	ORDER_FAILURE_REASON_REQUIRED_CODE = "ORDER_FAILURE_REASON_REQUIRED"
	ORDER_NOT_CANCELLABLE_CODE         = "ORDER_NOT_CANCELLABLE"
	ORDER_ADDRESS_NOT_FOUND_CODE       = "ORDER_ADDRESS_NOT_FOUND"
	ORDER_INVALID_FULFILLMENT_CODE     = "ORDER_INVALID_FULFILLMENT_TYPE"
)

const (
	ORDER_CART_NOT_ACTIVE_MSG         = "Cart is not active"
	ORDER_CART_EMPTY_MSG              = "Cart is empty"
	ORDER_CART_ALREADY_CHECKOUT_MSG   = "Cart is already in checkout"
	ORDER_NOT_FOUND_MSG               = "Order not found"
	ORDER_INVALID_STATUS_MSG          = "Invalid order status"
	ORDER_INVALID_TRANSITION_MSG      = "Invalid status transition from %s to %s"
	ORDER_TRANSACTION_ID_REQUIRED_MSG = "transactionId is required when status is confirmed"
	ORDER_FAILURE_REASON_REQUIRED_MSG = "failureReason is required when status is failed"
	ORDER_NOT_CANCELLABLE_MSG         = "Order is not in a cancellable state"
	ORDER_ADDRESS_NOT_FOUND_MSG       = "Address not found"
	ORDER_INVALID_FULFILLMENT_MSG     = "Invalid fulfillment type"
)

var (
	ErrCartNotActive = &commonError.AppError{
		Code:       ORDER_CART_NOT_ACTIVE_CODE,
		Message:    ORDER_CART_NOT_ACTIVE_MSG,
		StatusCode: http.StatusConflict,
	}

	ErrCartEmpty = &commonError.AppError{
		Code:       ORDER_CART_EMPTY_CODE,
		Message:    ORDER_CART_EMPTY_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCartAlreadyInCheckout = &commonError.AppError{
		Code:       ORDER_CART_ALREADY_CHECKOUT_CODE,
		Message:    ORDER_CART_ALREADY_CHECKOUT_MSG,
		StatusCode: http.StatusConflict,
	}

	ErrOrderNotFound = &commonError.AppError{
		Code:       ORDER_NOT_FOUND_CODE,
		Message:    ORDER_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	ErrInvalidOrderStatus = &commonError.AppError{
		Code:       ORDER_INVALID_STATUS_CODE,
		Message:    ORDER_INVALID_STATUS_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrTransactionIDRequired = &commonError.AppError{
		Code:       ORDER_TRANSACTION_ID_REQUIRED_CODE,
		Message:    ORDER_TRANSACTION_ID_REQUIRED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrFailureReasonRequired = &commonError.AppError{
		Code:       ORDER_FAILURE_REASON_REQUIRED_CODE,
		Message:    ORDER_FAILURE_REASON_REQUIRED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrOrderNotCancellable = &commonError.AppError{
		Code:       ORDER_NOT_CANCELLABLE_CODE,
		Message:    ORDER_NOT_CANCELLABLE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrAddressNotFound = &commonError.AppError{
		Code:       ORDER_ADDRESS_NOT_FOUND_CODE,
		Message:    ORDER_ADDRESS_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	ErrInvalidFulfillmentType = &commonError.AppError{
		Code:       ORDER_INVALID_FULFILLMENT_CODE,
		Message:    ORDER_INVALID_FULFILLMENT_MSG,
		StatusCode: http.StatusBadRequest,
	}
)

func ErrInvalidStatusTransition(from, to string) *commonError.AppError {
	return &commonError.AppError{
		Code:       ORDER_INVALID_TRANSITION_CODE,
		Message:    fmt.Sprintf(ORDER_INVALID_TRANSITION_MSG, from, to),
		StatusCode: http.StatusBadRequest,
	}
}
