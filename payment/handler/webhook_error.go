package handler

import (
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
