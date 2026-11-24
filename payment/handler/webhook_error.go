package handler

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/payment/utils/constant"
)

// isWebhookAuthError reports whether an error is a webhook signature failure.
func isWebhookAuthError(err error) bool {
	appErr, ok := commonError.AsAppError(err)
	if !ok {
		return false
	}
	return appErr.Code == constant.INVALID_WEBHOOK_SIGNATURE_CODE
}

// isWebhookNotFoundError reports routing failures (unknown gateway code).
// The contract requires 404 here — unlike apply failures, which stay 200 so
// the provider stops retrying, a wrong address must stay loud.
func isWebhookNotFoundError(err error) bool {
	appErr, ok := commonError.AsAppError(err)
	if !ok {
		return false
	}
	return appErr.StatusCode == http.StatusNotFound
}
