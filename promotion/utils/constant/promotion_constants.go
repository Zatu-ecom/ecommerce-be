package constant

// Promotion success messages
const (
	PROMOTION_PRODUCTS_ADDED_MSG       = "Products added to promotion successfully"
	PROMOTION_PRODUCTS_REMOVED_MSG     = "Products removed from promotion successfully"
	PROMOTION_ALL_PRODUCTS_REMOVED_MSG = "All products removed from promotion successfully"
	PROMOTION_PRODUCTS_RETRIEVED_MSG   = "Promotion products retrieved successfully"

	PROMOTION_CREATED_MSG        = "Promotion created successfully"
	PROMOTION_RETRIEVED_MSG      = "Promotion retrieved successfully"
	PROMOTIONS_LISTED_MSG        = "Promotions retrieved successfully"
	PROMOTION_UPDATED_MSG        = "Promotion updated successfully"
	PROMOTION_STATUS_UPDATED_MSG = "Promotion status updated successfully"
	PROMOTION_DELETED_MSG        = "Promotion deleted successfully"

	PROMOTION_VARIANTS_ADDED_MSG         = "Variants added to promotion successfully"
	PROMOTION_VARIANTS_REMOVED_MSG       = "Variants removed from promotion successfully"
	PROMOTION_ALL_VARIANTS_REMOVED_MSG   = "All variants removed from promotion successfully"
	PROMOTION_VARIANTS_RETRIEVED_MSG     = "Promotion variants retrieved successfully"
	PROMOTION_CATEGORIES_ADDED_MSG       = "Categories added to promotion successfully"
	PROMOTION_CATEGORIES_REMOVED_MSG     = "Categories removed from promotion successfully"
	PROMOTION_ALL_CATEGORIES_REMOVED_MSG = "All categories removed from promotion successfully"
	PROMOTION_CATEGORIES_RETRIEVED_MSG   = "Promotion categories retrieved successfully"

	PROMOTION_COLLECTIONS_ADDED_MSG       = "Collections added to promotion successfully"
	PROMOTION_COLLECTIONS_REMOVED_MSG     = "Collections removed from promotion successfully"
	PROMOTION_ALL_COLLECTIONS_REMOVED_MSG = "All collections removed from promotion successfully"
	PROMOTION_COLLECTIONS_RETRIEVED_MSG   = "Promotion collections retrieved successfully"
)

// Promotion failure messages
const (
	FAILED_TO_ADD_PROMOTION_PRODUCTS_MSG        = "Failed to add products to promotion"
	FAILED_TO_REMOVE_PROMOTION_PRODUCTS_MSG     = "Failed to remove products from promotion"
	FAILED_TO_REMOVE_ALL_PROMOTION_PRODUCTS_MSG = "Failed to remove all products from promotion"
	FAILED_TO_GET_PROMOTION_PRODUCTS_MSG        = "Failed to get promotion products"

	FAILED_TO_CREATE_PROMOTION_MSG        = "Failed to create promotion"
	FAILED_TO_GET_PROMOTION_MSG           = "Failed to retrieve promotion"
	FAILED_TO_LIST_PROMOTIONS_MSG         = "Failed to list promotions"
	FAILED_TO_UPDATE_PROMOTION_MSG        = "Failed to update promotion"
	FAILED_TO_UPDATE_PROMOTION_STATUS_MSG = "Failed to update promotion status"
	FAILED_TO_DELETE_PROMOTION_MSG        = "Failed to delete promotion"

	FAILED_TO_ADD_PROMOTION_VARIANTS_MSG          = "Failed to add variants to promotion"
	FAILED_TO_REMOVE_PROMOTION_VARIANTS_MSG       = "Failed to remove variants from promotion"
	FAILED_TO_REMOVE_ALL_PROMOTION_VARIANTS_MSG   = "Failed to remove all variants from promotion"
	FAILED_TO_GET_PROMOTION_VARIANTS_MSG          = "Failed to get promotion variants"
	FAILED_TO_ADD_PROMOTION_CATEGORIES_MSG        = "Failed to add categories to promotion"
	FAILED_TO_REMOVE_PROMOTION_CATEGORIES_MSG     = "Failed to remove categories from promotion"
	FAILED_TO_REMOVE_ALL_PROMOTION_CATEGORIES_MSG = "Failed to remove all categories from promotion"
	FAILED_TO_GET_PROMOTION_CATEGORIES_MSG        = "Failed to get promotion categories"

	FAILED_TO_ADD_PROMOTION_COLLECTIONS_MSG        = "Failed to add collections to promotion"
	FAILED_TO_REMOVE_PROMOTION_COLLECTIONS_MSG     = "Failed to remove collections from promotion"
	FAILED_TO_REMOVE_ALL_PROMOTION_COLLECTIONS_MSG = "Failed to remove all collections from promotion"
	FAILED_TO_GET_PROMOTION_COLLECTIONS_MSG        = "Failed to get promotion collections"

	INVALID_PROMOTION_ID_MSG = "Invalid promotion ID"
)

// Promotion validation reasons (used when filtering/skipping promotions)
const (
	VALIDATION_PROMOTION_NOT_ACTIVE_MSG            = "Promotion is not active"
	VALIDATION_PROMOTION_NOT_STARTED_MSG           = "Promotion has not started yet"
	VALIDATION_PROMOTION_ENDED_MSG                 = "Promotion has ended"
	VALIDATION_PROMOTION_USAGE_LIMIT_REACHED_MSG   = "Promotion usage limit reached"
	VALIDATION_UNABLE_TO_VERIFY_CUSTOMER_USAGE_MSG = "Unable to verify customer usage limit"
	VALIDATION_CUSTOMER_USAGE_LIMIT_REACHED_MSG    = "Customer usage limit reached for this promotion"
	VALIDATION_CUSTOMER_NOT_ELIGIBLE_MSG           = "Customer is not eligible for this promotion"
	VALIDATION_NON_STACKABLE_ALREADY_APPLIED_MSG   = "Non-stackable promotion cannot be applied as another promotion is already applied"
)

// Promotion field names
const (
	PROMOTION_FIELD             = "promotion"
	PROMOTIONS_FIELD            = "promotions"
	PROMOTION_PRODUCTS_FIELD    = "products"
	PROMOTION_VARIANTS_FIELD    = "variants"
	PROMOTION_CATEGORIES_FIELD  = "categories"
	PROMOTION_COLLECTIONS_FIELD = "collections"
)
