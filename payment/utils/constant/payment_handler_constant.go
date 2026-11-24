package constant

// Payment handler success messages.
const (
	PAYMENT_INITIATED_MSG      = "Payment initiated"
	PAYMENT_STATUS_FETCHED_MSG = "Payment status fetched"
	TRANSACTIONS_FETCHED_MSG   = "Transactions fetched"
	REFUND_INITIATED_MSG       = "Refund initiated"
	GATEWAYS_FETCHED_MSG       = "Gateways fetched"
	GATEWAY_FETCHED_MSG        = "Gateway fetched"
	GATEWAY_CONFIGURED_MSG     = "Gateway configured"
	GATEWAY_DEACTIVATED_MSG    = "Gateway deactivated"
	GATEWAY_TESTED_MSG         = "Credentials accepted"
	WEBHOOK_LOGS_FETCHED_MSG   = "Webhook logs fetched"
	WEBHOOK_RECEIVED_MSG       = "received"
)

// Payment handler error fallback messages (used only for unexpected errors).
const (
	FAILED_TO_INITIATE_PAYMENT_MSG   = "Failed to initiate payment"
	FAILED_TO_GET_PAYMENT_STATUS_MSG = "Failed to get payment status"
	FAILED_TO_LIST_TRANSACTIONS_MSG  = "Failed to list transactions"
	FAILED_TO_REFUND_PAYMENT_MSG     = "Failed to refund payment"
	FAILED_TO_LIST_GATEWAYS_MSG      = "Failed to list gateways"
	FAILED_TO_GET_GATEWAY_MSG        = "Failed to get gateway"
	FAILED_TO_CONFIGURE_GATEWAY_MSG  = "Failed to configure gateway"
	FAILED_TO_DEACTIVATE_GATEWAY_MSG = "Failed to deactivate gateway"
	FAILED_TO_TEST_GATEWAY_MSG       = "Failed to test gateway"
	FAILED_TO_LIST_WEBHOOK_LOGS_MSG  = "Failed to list webhook logs"
	FAILED_TO_READ_WEBHOOK_BODY_MSG  = "Failed to read webhook body"
)

// Route path params.
const (
	PARAM_CODE           = "code"
	PARAM_TRANSACTION_ID = "transactionId"
)

// Query params.
const (
	QUERY_PAGE        = "page"
	QUERY_PAGE_SIZE   = "pageSize"
	QUERY_ENVIRONMENT = "environment"
	QUERY_STATUS      = "status"
	QUERY_EVENT_TYPE  = "eventType"
)
