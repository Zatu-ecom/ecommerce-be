package utils

// Collection error codes
const (
	COLLECTION_NOT_FOUND_CODE            = "COLLECTION_NOT_FOUND"
	COLLECTION_EXISTS_CODE               = "COLLECTION_EXISTS"
	UNAUTHORIZED_COLLECTION_ACCESS_CODE  = "UNAUTHORIZED_COLLECTION_ACCESS"
	PRODUCT_NOT_IN_COLLECTION_CODE       = "PRODUCT_NOT_IN_COLLECTION"
	INVALID_COLLECTION_PRODUCT_CODE      = "INVALID_COLLECTION_PRODUCT"
	COLLECTION_INVALID_FILE_CODE         = "COLLECTION_INVALID_FILE"
)

// Collection messages
const (
	COLLECTION_NOT_FOUND_MSG           = "Collection not found"
	COLLECTION_EXISTS_MSG              = "Collection with this slug already exists for this seller"
	UNAUTHORIZED_COLLECTION_ACCESS_MSG = "You do not have permission to access this collection"
	PRODUCT_NOT_IN_COLLECTION_MSG      = "One or more products are not in this collection"
	INVALID_COLLECTION_PRODUCT_MSG     = "One or more products are invalid or do not belong to this seller"
	COLLECTION_INVALID_FILE_MSG        = "Collection image file is invalid or not accessible"
)

// Collection success messages
const (
	COLLECTION_CREATED_MSG              = "Collection created successfully"
	COLLECTION_UPDATED_MSG              = "Collection updated successfully"
	COLLECTION_DELETED_MSG              = "Collection deleted successfully"
	COLLECTIONS_RETRIEVED_MSG           = "Collections retrieved successfully"
	COLLECTION_PRODUCTS_ADDED_MSG       = "Products added to collection successfully"
	COLLECTION_PRODUCTS_REMOVED_MSG     = "Products removed from collection successfully"
	COLLECTION_PRODUCTS_RETRIEVED_MSG   = "Collection products retrieved successfully"
	COLLECTION_PRODUCTS_REORDERED_MSG   = "Collection products reordered successfully"
)

// Collection operation failure messages
const (
	FAILED_TO_CREATE_COLLECTION_MSG        = "Failed to create collection"
	FAILED_TO_UPDATE_COLLECTION_MSG        = "Failed to update collection"
	FAILED_TO_DELETE_COLLECTION_MSG        = "Failed to delete collection"
	FAILED_TO_GET_COLLECTIONS_MSG          = "Failed to get collections"
	FAILED_TO_ADD_COLLECTION_PRODUCTS_MSG  = "Failed to add products to collection"
	FAILED_TO_REMOVE_COLLECTION_PRODUCTS_MSG = "Failed to remove products from collection"
	FAILED_TO_GET_COLLECTION_PRODUCTS_MSG  = "Failed to get collection products"
	FAILED_TO_REORDER_COLLECTION_PRODUCTS_MSG = "Failed to reorder collection products"
)

// Collection field names
const (
	COLLECTION_FIELD_NAME  = "collection"
	COLLECTIONS_FIELD_NAME = "collections"
)
