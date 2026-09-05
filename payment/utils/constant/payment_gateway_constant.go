package constant

const (
	PAYMENT_GATEWAY_NOT_FOUND_MESSAGE     = "Payment gateway not found"
	PAYMENT_GATEWAY_NOT_ACTIVE_MESSAGE    = "Payment gateway not active"
	PAYMENT_GATEWAY_NOT_SUPPORTED_MESSAGE = "Payment gateway not supported"

	PAYMENT_TRANSACTION_NOT_FOUND_MESSAGE = "Payment transaction not found"
	PAYMENT_ORDER_NOT_FOUND_MESSAGE       = "Order not found for payment"
	REFUND_NOT_FOUND_MESSAGE              = "Refund not found"
	DUPLICATE_PAYMENT_MESSAGE             = "A payment already exists for this order"
	GATEWAY_VALIDATION_MESSAGE            = "Invalid gateway credentials"
	INVALID_WEBHOOK_SIGNATURE_MESSAGE     = "Invalid webhook signature"
	DUPLICATE_WEBHOOK_MESSAGE             = "Duplicate webhook event ignored"
	REFUND_NOT_ALLOWED_MESSAGE            = "Refund not allowed for this payment"
	GATEWAY_NOT_CONFIGURED_MESSAGE        = "Payment gateway is not configured for this seller"
	GATEWAY_UNSUPPORTED_CURRENCY_MESSAGE  = "Payment gateway does not support the seller's country or currency"
)

// Gateway code constants
const (
	GATEWAY_CODE_RAZORPAY = "razorpay"
)

// Payment transaction status constants
const (
	TRANSACTION_STATUS_PENDING            = "pending"
	TRANSACTION_STATUS_COMPLETED          = "completed"
	TRANSACTION_STATUS_FAILED             = "failed"
	TRANSACTION_STATUS_REFUNDED           = "refunded"
	TRANSACTION_STATUS_PARTIALLY_REFUNDED = "partially_refunded"
)

// Payment transaction reference types
const (
	REFERENCE_TYPE_ORDER        = "order"
	REFERENCE_TYPE_SUBSCRIPTION = "subscription"
)

// Payment transaction event types
const (
	EVENT_TYPE_INITIATED               = "initiated"
	EVENT_TYPE_GATEWAY_SESSION_CREATED = "gateway_session_created"
	EVENT_TYPE_AUTHORIZED              = "authorized"
	EVENT_TYPE_CAPTURED                = "captured"
	EVENT_TYPE_COMPLETED               = "completed"
	EVENT_TYPE_FAILED                  = "failed"
	EVENT_TYPE_REFUND_INITIATED        = "refund_initiated"
	EVENT_TYPE_REFUND_COMPLETED        = "refund_completed"
	EVENT_TYPE_REFUND_FAILED           = "refund_failed"
)

// Event sources
const (
	EVENT_SOURCE_API     = "api"
	EVENT_SOURCE_WEBHOOK = "webhook"
	EVENT_SOURCE_SYSTEM  = "system"
	EVENT_SOURCE_ADMIN   = "admin"
)

// Razorpay webhook events
const (
	RAZORPAY_EVENT_PAYMENT_AUTHORIZED = "payment.authorized"
	RAZORPAY_EVENT_PAYMENT_CAPTURED   = "payment.captured"
	RAZORPAY_EVENT_PAYMENT_FAILED     = "payment.failed"
	RAZORPAY_EVENT_REFUND_CREATED     = "refund.created"
	RAZORPAY_EVENT_REFUND_PROCESSED   = "refund.processed"
	RAZORPAY_EVENT_REFUND_FAILED      = "refund.failed"
)

// Razorpay API base URL
const (
	RAZORPAY_BASE_URL = "https://api.razorpay.com/v1"
)
