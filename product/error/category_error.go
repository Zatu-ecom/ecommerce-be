package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/utils"
)

// Category Errors

var (
	// ErrCategoryExists is returned when a category with the same name already exists
	ErrCategoryExists = &commonError.AppError{
		Code:       utils.CATEGORY_EXISTS_CODE,
		Message:    utils.CATEGORY_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrCategoryNotFound is returned when a category is not found
	ErrCategoryNotFound = &commonError.AppError{
		Code:       utils.CATEGORY_NOT_FOUND_CODE,
		Message:    utils.CATEGORY_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrCategoryHasProducts is returned when trying to delete a category with products
	ErrCategoryHasProducts = &commonError.AppError{
		Code:       utils.CATEGORY_HAS_PRODUCTS_CODE,
		Message:    utils.CATEGORY_HAS_PRODUCTS_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrCategoryHasChildren is returned when trying to delete a category with child categories
	ErrCategoryHasChildren = &commonError.AppError{
		Code:       utils.CATEGORY_HAS_CHILDREN_CODE,
		Message:    utils.CATEGORY_HAS_CHILDREN_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrInvalidParentCategory is returned when parent category is invalid
	ErrInvalidParentCategory = &commonError.AppError{
		Code:       utils.INVALID_PARENT_CATEGORY_CODE,
		Message:    utils.INVALID_PARENT_CATEGORY_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrUnauthorizedCategoryUpdate = &commonError.AppError{
		Code:       utils.UNAUTHORIZED_CATEGORY_UPDATE_CODE,
		Message:    utils.UNAUTHORIZED_CATEGORY_UPDATE_MSG,
		StatusCode: http.StatusForbidden,
	}

	// ErrAttributeAlreadyLinked is returned when attribute is already linked to category
	ErrAttributeAlreadyLinked = &commonError.AppError{
		Code:       "ATTRIBUTE_ALREADY_LINKED",
		Message:    "Attribute is already linked to this category",
		StatusCode: http.StatusConflict,
	}

	// ErrAttributeNotLinked is returned when trying to unlink attribute that is not linked
	ErrAttributeNotLinked = &commonError.AppError{
		Code:       "ATTRIBUTE_NOT_LINKED",
		Message:    "Attribute is not linked to this category",
		StatusCode: http.StatusNotFound,
	}
)
