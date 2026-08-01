package constant

import "ecommerce-be/common/constants"

// Discount code success messages
const (
	DISCOUNT_CODE_CREATED_MSG        = "Discount code created successfully"
	DISCOUNT_CODE_RETRIEVED_MSG      = "Discount code retrieved successfully"
	DISCOUNT_CODES_LISTED_MSG        = "Discount codes retrieved successfully"
	DISCOUNT_CODE_UPDATED_MSG        = "Discount code updated successfully"
	DISCOUNT_CODE_STATUS_UPDATED_MSG = "Discount code status updated successfully"
	DISCOUNT_CODE_DELETED_MSG        = "Discount code deleted successfully"

	DISCOUNT_CODE_PRODUCTS_ADDED_MSG       = "Products added to discount code successfully"
	DISCOUNT_CODE_PRODUCTS_REMOVED_MSG     = "Products removed from discount code successfully"
	DISCOUNT_CODE_ALL_PRODUCTS_REMOVED_MSG = "All products removed from discount code successfully"
	DISCOUNT_CODE_PRODUCTS_RETRIEVED_MSG   = "Discount code products retrieved successfully"

	DISCOUNT_CODE_VARIANTS_ADDED_MSG       = "Variants added to discount code successfully"
	DISCOUNT_CODE_VARIANTS_REMOVED_MSG     = "Variants removed from discount code successfully"
	DISCOUNT_CODE_ALL_VARIANTS_REMOVED_MSG = "All variants removed from discount code successfully"
	DISCOUNT_CODE_VARIANTS_RETRIEVED_MSG   = "Discount code variants retrieved successfully"

	DISCOUNT_CODE_CATEGORIES_ADDED_MSG       = "Categories added to discount code successfully"
	DISCOUNT_CODE_CATEGORIES_REMOVED_MSG     = "Categories removed from discount code successfully"
	DISCOUNT_CODE_ALL_CATEGORIES_REMOVED_MSG = "All categories removed from discount code successfully"
	DISCOUNT_CODE_CATEGORIES_RETRIEVED_MSG   = "Discount code categories retrieved successfully"

	DISCOUNT_CODE_COLLECTIONS_ADDED_MSG       = "Collections added to discount code successfully"
	DISCOUNT_CODE_COLLECTIONS_REMOVED_MSG     = "Collections removed from discount code successfully"
	DISCOUNT_CODE_ALL_COLLECTIONS_REMOVED_MSG = "All collections removed from discount code successfully"
	DISCOUNT_CODE_COLLECTIONS_RETRIEVED_MSG   = "Discount code collections retrieved successfully"
)

// Discount code failure messages
const (
	FAILED_TO_CREATE_DISCOUNT_CODE_MSG        = "Failed to create discount code"
	FAILED_TO_GET_DISCOUNT_CODE_MSG           = "Failed to retrieve discount code"
	FAILED_TO_LIST_DISCOUNT_CODES_MSG         = "Failed to list discount codes"
	FAILED_TO_UPDATE_DISCOUNT_CODE_MSG        = "Failed to update discount code"
	FAILED_TO_UPDATE_DISCOUNT_CODE_STATUS_MSG = "Failed to update discount code status"
	FAILED_TO_DELETE_DISCOUNT_CODE_MSG        = "Failed to delete discount code"
	INVALID_DISCOUNT_CODE_ID_MSG              = "Invalid discount code ID"

	FAILED_TO_ADD_DISCOUNT_CODE_PRODUCTS_MSG        = "Failed to add products to discount code"
	FAILED_TO_REMOVE_DISCOUNT_CODE_PRODUCTS_MSG     = "Failed to remove products from discount code"
	FAILED_TO_REMOVE_ALL_DISCOUNT_CODE_PRODUCTS_MSG = "Failed to remove all products from discount code"
	FAILED_TO_GET_DISCOUNT_CODE_PRODUCTS_MSG        = "Failed to get discount code products"

	FAILED_TO_ADD_DISCOUNT_CODE_VARIANTS_MSG        = "Failed to add variants to discount code"
	FAILED_TO_REMOVE_DISCOUNT_CODE_VARIANTS_MSG     = "Failed to remove variants from discount code"
	FAILED_TO_REMOVE_ALL_DISCOUNT_CODE_VARIANTS_MSG = "Failed to remove all variants from discount code"
	FAILED_TO_GET_DISCOUNT_CODE_VARIANTS_MSG        = "Failed to get discount code variants"

	FAILED_TO_ADD_DISCOUNT_CODE_CATEGORIES_MSG        = "Failed to add categories to discount code"
	FAILED_TO_REMOVE_DISCOUNT_CODE_CATEGORIES_MSG     = "Failed to remove categories from discount code"
	FAILED_TO_REMOVE_ALL_DISCOUNT_CODE_CATEGORIES_MSG = "Failed to remove all categories from discount code"
	FAILED_TO_GET_DISCOUNT_CODE_CATEGORIES_MSG        = "Failed to get discount code categories"

	FAILED_TO_ADD_DISCOUNT_CODE_COLLECTIONS_MSG        = "Failed to add collections to discount code"
	FAILED_TO_REMOVE_DISCOUNT_CODE_COLLECTIONS_MSG     = "Failed to remove collections from discount code"
	FAILED_TO_REMOVE_ALL_DISCOUNT_CODE_COLLECTIONS_MSG = "Failed to remove all collections from discount code"
	FAILED_TO_GET_DISCOUNT_CODE_COLLECTIONS_MSG        = "Failed to get discount code collections"
)

// Discount code / coupon response field keys
const (
	DISCOUNT_CODE_FIELD             = "discountCode"
	DISCOUNT_CODES_FIELD            = "discountCodes"
	DISCOUNT_CODE_PRODUCTS_FIELD    = "products"
	DISCOUNT_CODE_VARIANTS_FIELD    = "variants"
	DISCOUNT_CODE_CATEGORIES_FIELD  = "categories"
	DISCOUNT_CODE_COLLECTIONS_FIELD = "collections"
)

// AVAILABLE_COUPONS_LIST_LIMIT caps storefront available-coupon evaluation (cart embed + dedicated GET).
const AVAILABLE_COUPONS_LIST_LIMIT = 50

// Discount Code (coupon definition) Base Path — under promotion module
const API_BASE_DISCOUNT_CODE = constants.APIBasePromotion + "/discount-code"
