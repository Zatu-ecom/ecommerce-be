package constant

// Inventory success messages
const (
	INVENTORY_UPDATED_MSG             = "Inventory updated successfully"
	INVENTORY_ADJUSTED_MSG            = "Inventory adjusted successfully" // Deprecated: use INVENTORY_UPDATED_MSG
	INVENTORY_RETRIEVED_MSG           = "Inventory retrieved successfully"
	INVENTORIES_RETRIEVED_MSG         = "Inventories retrieved successfully"
	INVENTORY_TRANSACTION_CREATED_MSG = "Inventory transaction created successfully"
)

// Inventory error messages
const (
	INVENTORY_NOT_FOUND_MSG          = "Inventory not found"
	INSUFFICIENT_STOCK_MSG           = "Insufficient stock available"
	INVALID_QUANTITY_MSG             = "Invalid quantity"
	NEGATIVE_STOCK_MSG               = "Operation would result in negative stock"
	BELOW_THRESHOLD_MSG              = "Operation would result in quantity below threshold. Adjust threshold if backorder is allowed"
	INSUFFICIENT_RESERVED_STOCK_MSG  = "Insufficient reserved stock to release"
	VARIANT_NOT_FOUND_MSG            = "Product variant not found"
	INVALID_TRANSACTION_TYPE_MSG     = "Invalid transaction type"
	INVALID_ADJUSTMENT_TYPE_MSG      = "Invalid adjustment type. Must be ADD or REMOVE"
	DIRECTION_REQUIRED_MSG          = "Direction is required for ADJUSTMENT type"
	DIRECTION_NOT_ALLOWED_MSG       = "Direction is not allowed for this transaction type"
	NOT_MANUAL_TRANSACTION_MSG      = "Transaction type not allowed for adjust API"
	REFERENCE_REQUIRED_MSG          = "Reference ID is required for this transaction type (Order ID, PO Number, Transfer ID, etc.)"
)

// Inventory operation failure messages
const (
	FAILED_TO_ADJUST_INVENTORY_MSG  = "Failed to adjust inventory"
	FAILED_TO_GET_INVENTORY_MSG     = "Failed to get inventory"
	FAILED_TO_CREATE_TRANSACTION_MSG = "Failed to create inventory transaction"
)

// Inventory field names
const (
	INVENTORY_FIELD_NAME   = "inventory"
	INVENTORIES_FIELD_NAME = "inventories"
	TRANSACTION_FIELD_NAME = "transaction"
)

// Inventory validation messages
const (
	VARIANT_ID_REQUIRED_MSG      = "Variant ID is required"
	LOCATION_ID_REQUIRED_MSG     = "Location ID is required"
	QUANTITY_REQUIRED_MSG        = "Quantity is required"
	QUANTITY_POSITIVE_MSG        = "Quantity must be greater than 0"
	TRANSACTION_TYPE_REQUIRED_MSG = "Transaction type is required"
	REASON_REQUIRED_MSG          = "Reason is required"
	REASON_LENGTH_MSG            = "Reason must be between 5 and 500 characters"
)
