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
)
