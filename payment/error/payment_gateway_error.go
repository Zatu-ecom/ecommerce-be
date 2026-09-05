package error

import (
	"net/http"

	"ecommerce-be/common/error"
	"ecommerce-be/payment/utils/constant"
)

var (
	ErrorPaymentGatewayNotFound = &error.AppError{
		Code:       constant.PAYMENT_GATEWAY_NOT_FOUND_CODE,
		Message:    constant.PAYMENT_GATEWAY_NOT_FOUND_MESSAGE,
		StatusCode: http.StatusNotFound,
	}

	ErrorPaymentGatewayNotActive = &error.AppError{
		Code:       constant.PAYMENT_GATEWAY_NOT_ACTIVE_CODE,
		Message:    constant.PAYMENT_GATEWAY_NOT_ACTIVE_MESSAGE,
		StatusCode: http.StatusNotFound,
	}

	ErrorPaymentGatewayNotSupported = &error.AppError{
		Code:       constant.PAYMENT_GATEWAY_NOT_SUPPORTED_CODE,
		Message:    constant.PAYMENT_GATEWAY_NOT_SUPPORTED_MESSAGE,
		StatusCode: http.StatusNotFound,
	}

	ErrorPaymentTransactionNotFound = &error.AppError{
		Code:       constant.PAYMENT_TRANSACTION_NOT_FOUND_CODE,
		Message:    constant.PAYMENT_TRANSACTION_NOT_FOUND_MESSAGE,
		StatusCode: http.StatusNotFound,
	}

	ErrorPaymentOrderNotFound = &error.AppError{
		Code:       constant.PAYMENT_ORDER_NOT_FOUND_CODE,
		Message:    constant.PAYMENT_ORDER_NOT_FOUND_MESSAGE,
		StatusCode: http.StatusNotFound,
	}

	ErrorRefundNotFound = &error.AppError{
		Code:       constant.REFUND_NOT_FOUND_CODE,
		Message:    constant.REFUND_NOT_FOUND_MESSAGE,
		StatusCode: http.StatusNotFound,
	}

	ErrorDuplicatePayment = &error.AppError{
		Code:       constant.DUPLICATE_PAYMENT_CODE,
		Message:    constant.DUPLICATE_PAYMENT_MESSAGE,
		StatusCode: http.StatusConflict,
	}

	ErrorGatewayValidation = &error.AppError{
		Code:       constant.GATEWAY_VALIDATION_CODE,
		Message:    constant.GATEWAY_VALIDATION_MESSAGE,
		StatusCode: http.StatusBadRequest,
	}

	ErrorInvalidWebhookSignature = &error.AppError{
		Code:       constant.INVALID_WEBHOOK_SIGNATURE_CODE,
		Message:    constant.INVALID_WEBHOOK_SIGNATURE_MESSAGE,
		StatusCode: http.StatusUnauthorized,
	}

	ErrorDuplicateWebhook = &error.AppError{
		Code:       constant.DUPLICATE_WEBHOOK_CODE,
		Message:    constant.DUPLICATE_WEBHOOK_MESSAGE,
		StatusCode: http.StatusOK,
	}

	ErrorRefundNotAllowed = &error.AppError{
		Code:       constant.REFUND_NOT_ALLOWED_CODE,
		Message:    constant.REFUND_NOT_ALLOWED_MESSAGE,
		StatusCode: http.StatusBadRequest,
	}

	ErrorGatewayNotConfigured = &error.AppError{
		Code:       constant.GATEWAY_NOT_CONFIGURED_CODE,
		Message:    constant.GATEWAY_NOT_CONFIGURED_MESSAGE,
		StatusCode: http.StatusBadRequest,
	}

	ErrorGatewayUnsupportedCurrency = &error.AppError{
		Code:       constant.GATEWAY_UNSUPPORTED_CURRENCY_CODE,
		Message:    constant.GATEWAY_UNSUPPORTED_CURRENCY_MESSAGE,
		StatusCode: http.StatusBadRequest,
	}
)
