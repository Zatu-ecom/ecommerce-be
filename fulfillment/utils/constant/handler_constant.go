package constant

// Route params and handler success messages (no magic strings in handlers).
const (
	PARAM_CODE = "code"
)

const (
	COURIERS_FETCHED_MSG    = "Couriers fetched"
	COURIER_FETCHED_MSG     = "Courier fetched"
	COURIER_CONFIGURED_MSG  = "Courier configured"
	COURIER_TESTED_MSG      = "Credentials accepted"
	FAILED_TO_LIST_MSG      = "Failed to list couriers"
	FAILED_TO_GET_MSG       = "Failed to fetch courier"
	FAILED_TO_CONFIGURE_MSG = "Failed to configure courier"
	FAILED_TO_TEST_MSG      = "Failed to test courier credentials"
)

const (
	SHIPMENTS_PLANNED_MSG     = "Shipments planned"
	SHIPMENT_CREATED_MSG      = "Shipment draft created"
	SHIPMENT_FETCHED_MSG      = "Shipment fetched"
	SHIPMENTS_FETCHED_MSG     = "Shipments fetched"
	SHIPMENT_UPDATED_MSG      = "Shipment draft updated"
	FAILED_TO_PLAN_MSG        = "Failed to plan shipments"
	FAILED_TO_CREATE_MSG      = "Failed to create shipment draft"
	FAILED_TO_LIST_SHIP_MSG   = "Failed to list shipments"
	FAILED_TO_GET_SHIP_MSG    = "Failed to fetch shipment"
	FAILED_TO_UPDATE_SHIP_MSG = "Failed to update shipment draft"
)

// Booking + post-booking messages.
const (
	SHIPMENT_BOOKED_MSG           = "Shipment booked"
	SHIPMENTS_BOOKED_MSG          = "Shipments booked"
	PICKUP_SCHEDULED_MSG          = "Pickup scheduled"
	SHIPMENT_CANCELLED_MSG        = "Shipment cancelled"
	ADDRESS_CONFIRMED_MSG         = "Delivery address confirmed"
	RATES_FETCHED_MSG             = "Rates fetched"
	FAILED_TO_BOOK_MSG            = "Failed to book shipment"
	FAILED_TO_PICKUP_MSG          = "Failed to schedule pickup"
	FAILED_TO_CANCEL_MSG          = "Failed to cancel shipment"
	FAILED_TO_LABEL_MSG           = "Failed to fetch shipping label"
	FAILED_TO_CONFIRM_ADDRESS_MSG = "Failed to confirm address"
	FAILED_TO_RATE_MSG            = "Failed to fetch rates"
	// LABEL_FAILED_CODE mirrors FULFILLMENT_LABEL_FAILED_CODE (single source).
	LABEL_FAILED_CODE = FULFILLMENT_LABEL_FAILED_CODE
)

// NDR + return messages.
const (
	NDR_ACTED_MSG        = "NDR action recorded"
	RTO_REQUESTED_MSG    = "Return requested"
	RETURN_CREATED_MSG   = "Return shipment created"
	FAILED_TO_NDR_MSG    = "Failed to act on NDR"
	FAILED_TO_RTO_MSG    = "Failed to request return"
	FAILED_TO_RETURN_MSG = "Failed to create return"
)

// Tracking + webhook messages.
const (
	WEBHOOK_RECEIVED_MSG       = "Webhook received"
	TRACK_FETCHED_MSG          = "Tracking fetched"
	WEBHOOK_LOGS_FETCHED_MSG   = "Webhook logs fetched"
	FAILED_TO_WEBHOOK_MSG      = "Failed to process webhook"
	FAILED_TO_TRACK_MSG        = "Failed to fetch tracking"
	FAILED_TO_REFRESH_MSG      = "Failed to refresh tracking"
	FAILED_TO_WEBHOOK_LOGS_MSG = "Failed to fetch webhook logs"
)
