package utils

// Category messages
const (
	CATEGORY_EXISTS_MSG             = "Category with this name already exists in the same parent"
	CATEGORY_NOT_FOUND_MSG          = "Category not found"
	CATEGORY_HAS_PRODUCTS_MSG       = "Cannot delete category with active products"
	CATEGORY_HAS_CHILDREN_MSG       = "Cannot delete category with active child categories"
	INVALID_PARENT_CATEGORY_MSG     = "Invalid parent category"
	CATEGORY_NAME_REQUIRED_MSG      = "Category name is required"
	CATEGORY_NAME_LENGTH_MSG        = "Category name must be between 3 and 100 characters"
	CATEGORY_DESCRIPTION_LENGTH_MSG = "Category description must not exceed 500 characters"
)

// Attribute Definition messages
const (
	ATTRIBUTE_DEFINITION_EXISTS_MSG    = "Attribute definition with this key already exists"
	ATTRIBUTE_DEFINITION_NOT_FOUND_MSG = "Attribute definition not found"
	ATTRIBUTE_KEY_REQUIRED_MSG         = "Attribute key is required"
	ATTRIBUTE_KEY_LENGTH_MSG           = "Attribute key must be between 3 and 50 characters"
	ATTRIBUTE_KEY_FORMAT_MSG           = "Attribute key must contain only lowercase letters, numbers, and underscores"
	ATTRIBUTE_NAME_REQUIRED_MSG        = "Attribute name is required"
	ATTRIBUTE_NAME_LENGTH_MSG          = "Attribute name must be between 3 and 100 characters"
	ATTRIBUTE_DATA_TYPE_REQUIRED_MSG   = "Attribute data type is required"
	ATTRIBUTE_DATA_TYPE_INVALID_MSG    = "Invalid attribute data type. Must be string, number, boolean, or array"
	ATTRIBUTE_UNIT_LENGTH_MSG          = "Attribute unit must not exceed 20 characters"
	ATTRIBUTE_DESCRIPTION_LENGTH_MSG   = "Attribute description must not exceed 500 characters"
)

// Product messages
const (
	PRODUCT_EXISTS_MSG              = "Product with this SKU already exists"
	PRODUCT_NOT_FOUND_MSG           = "Product not found"
	PRODUCT_NAME_REQUIRED_MSG       = "Product name is required"
	PRODUCT_NAME_LENGTH_MSG         = "Product name must be between 3 and 200 characters"
	PRODUCT_CATEGORY_REQUIRED_MSG   = "Product category is required"
	PRODUCT_CATEGORY_INVALID_MSG    = "Invalid product category"
	PRODUCT_SKU_REQUIRED_MSG        = "Product SKU is required"
	PRODUCT_SKU_LENGTH_MSG          = "Product SKU must be between 3 and 50 characters"
	PRODUCT_SKU_UNIQUE_MSG          = "Product SKU must be unique"
	PRODUCT_PRICE_REQUIRED_MSG      = "Product price is required"
	PRODUCT_PRICE_POSITIVE_MSG      = "Product price must be positive"
	PRODUCT_CURRENCY_INVALID_MSG    = "Invalid currency code. Must be 3 characters"
	PRODUCT_DESCRIPTION_LENGTH_MSG  = "Product description must not exceed 500 characters"
	PRODUCT_LONG_DESC_LENGTH_MSG    = "Product long description must not exceed 5000 characters"
	PRODUCT_IMAGES_LIMIT_MSG        = "Product cannot have more than 10 images"
	PRODUCT_DISCOUNT_RANGE_MSG      = "Product discount must be between 0 and 100"
	PRODUCT_TAGS_LIMIT_MSG          = "Product cannot have more than 20 tags"
	PRODUCT_ATTRIBUTES_REQUIRED_MSG = "Product attributes are required based on category configuration"
)

// Package Option messages
const (
	PACKAGE_OPTION_NOT_FOUND_MSG      = "Package option not found"
	PACKAGE_OPTION_NAME_REQUIRED_MSG  = "Package option name is required"
	PACKAGE_OPTION_PRICE_REQUIRED_MSG = "Package option price is required"
	PACKAGE_OPTION_PRICE_POSITIVE_MSG = "Package option price must be positive"
)

// Operation failure messages
const (
	FAILED_TO_CREATE_CATEGORY_MSG          = "Failed to create category"
	FAILED_TO_UPDATE_CATEGORY_MSG          = "Failed to update category"
	FAILED_TO_DELETE_CATEGORY_MSG          = "Failed to delete category"
	FAILED_TO_GET_CATEGORIES_MSG           = "Failed to get categories"
	FAILED_TO_CREATE_ATTRIBUTE_MSG         = "Failed to create attribute definition"
	FAILED_TO_UPDATE_ATTRIBUTE_MSG         = "Failed to update attribute definition"
	FAILED_TO_GET_ATTRIBUTES_MSG           = "Failed to get attribute definitions"
	FAILED_TO_CONFIGURE_CATEGORY_ATTRS_MSG = "Failed to configure category attributes"
	FAILED_TO_GET_CATEGORY_ATTRS_MSG       = "Failed to get category attributes"
	FAILED_TO_CREATE_PRODUCT_MSG           = "Failed to create product"
	FAILED_TO_UPDATE_PRODUCT_MSG           = "Failed to update product"
	FAILED_TO_DELETE_PRODUCT_MSG           = "Failed to delete product"
	FAILED_TO_GET_PRODUCTS_MSG             = "Failed to get products"
	FAILED_TO_GET_PRODUCT_MSG              = "Failed to get product"
	FAILED_TO_UPDATE_STOCK_MSG             = "Failed to update product stock"
	FAILED_TO_SEARCH_PRODUCTS_MSG          = "Failed to search products"
	FAILED_TO_GET_FILTERS_MSG              = "Failed to get product filters"
	FAILED_TO_GET_RELATED_PRODUCTS_MSG     = "Failed to get related products"
	FAILED_TO_ADD_PACKAGE_OPTION_MSG       = "Failed to add package option"
	FAILED_TO_UPDATE_PACKAGE_OPTION_MSG    = "Failed to update package option"
	FAILED_TO_DELETE_PACKAGE_OPTION_MSG    = "Failed to delete package option"
	FAILED_TO_GET_CATEGORY_ATTRIBUTES_MSG  = "Failed to get category attributes"
)

// Permission and access messages
const (
	PERMISSION_DENIED_MSG        = "You don't have permission to perform this action"
	ADMIN_REQUIRED_MSG           = "Admin access required"
	SELLER_OR_ADMIN_REQUIRED_MSG = "Seller or Admin access required"
)

// Validation messages
const (
	VALIDATION_FAILED_MSG      = "Validation failed"
	INVALID_REQUEST_FORMAT_MSG = "Invalid request format"
)

// Business rule messages
const (
	CATEGORY_CANNOT_BE_DELETED_MSG  = "Category cannot be deleted due to business rules"
	PRODUCT_CANNOT_BE_DELETED_MSG   = "Product cannot be deleted due to business rules"
	ATTRIBUTE_VALIDATION_FAILED_MSG = "Attribute validation failed for this category"
)
