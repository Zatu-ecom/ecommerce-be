package error

import (
	"net/http"

	"ecommerce-be/common/error"
	fulfillmentconstant "ecommerce-be/fulfillment/utils/constant"
)

var (
	ErrorFulfillmentNotFound = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_NOT_FOUND_CODE,
		Message:    "Fulfillment shipment not found",
		StatusCode: http.StatusNotFound,
	}

	ErrorFulfillmentInvalidState = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_INVALID_STATE_CODE,
		Message:    "Operation not allowed in the current shipment status",
		StatusCode: http.StatusConflict,
	}

	ErrorProviderNotSupported = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_PROVIDER_NOT_SUPPORTED_CODE,
		Message:    "Courier provider not supported",
		StatusCode: http.StatusNotFound,
	}

	ErrorProviderNotConfigured = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_PROVIDER_NOT_CONFIGURED_CODE,
		Message:    "Courier provider is not configured for this seller",
		StatusCode: http.StatusConflict,
	}

	ErrorCredentialsInvalid = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_CREDENTIALS_INVALID_CODE,
		Message:    "Invalid courier credentials",
		StatusCode: http.StatusBadRequest,
	}

	ErrorEncryptionKeyMissing = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_ENCRYPTION_KEY_MISSING_CODE,
		Message:    "Courier credential encryption is not configured",
		StatusCode: http.StatusInternalServerError,
	}

	ErrorRateFailed = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_RATE_FAILED_CODE,
		Message:    "Failed to fetch courier rates",
		StatusCode: http.StatusBadGateway,
	}

	ErrorBookFailed = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_BOOK_FAILED_CODE,
		Message:    "Courier booking failed",
		StatusCode: http.StatusBadGateway,
	}

	// ErrorLabelFailed defaults to 400 (nothing to fetch yet). Transient
	// provider failures surface as plain errors; the label handler maps
	// those to 503 under the same code (api-contracts §7).
	ErrorLabelFailed = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_LABEL_FAILED_CODE,
		Message:    "Failed to fetch shipping label",
		StatusCode: http.StatusBadRequest,
	}

	ErrorCapabilityUnsupported = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_CAPABILITY_UNSUPPORTED_CODE,
		Message:    "Courier does not support this operation",
		StatusCode: http.StatusBadRequest,
	}

	ErrorWebhookUnverified = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_WEBHOOK_UNVERIFIED_CODE,
		Message:    "Courier webhook signature verification failed",
		StatusCode: http.StatusUnauthorized,
	}

	ErrorApplyMismatch = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_APPLY_MISMATCH_CODE,
		Message:    "Courier event does not match this shipment",
		StatusCode: http.StatusConflict,
	}

	ErrorStockMismatch = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_STOCK_MISMATCH_CODE,
		Message:    "Reserved stock no longer covers this shipment",
		StatusCode: http.StatusConflict,
	}

	ErrorWeightRequired = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_WEIGHT_REQUIRED_CODE,
		Message:    "Weight is required before booking",
		StatusCode: http.StatusBadRequest,
	}

	ErrorAddressChanged = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_ADDRESS_CHANGED_CODE,
		Message:    "Delivery address changed since planning; review required",
		StatusCode: http.StatusConflict,
	}

	ErrorAutoBookAmbiguous = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_AUTO_BOOK_AMBIGUOUS_CODE,
		Message:    "Auto-book cannot choose a courier; book manually or set a rate preference",
		StatusCode: http.StatusConflict,
	}

	// ErrorRateLimited throttles the public rate endpoint (generous
	// per-seller budget; counting via durable Lua, never cached).
	ErrorRateLimited = &error.AppError{
		Code:       fulfillmentconstant.FULFILLMENT_RATE_LIMITED_CODE,
		Message:    "Rate limit exceeded, retry shortly",
		StatusCode: http.StatusTooManyRequests,
	}
)
