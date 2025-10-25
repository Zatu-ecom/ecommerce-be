package errors

import (
	"net/http"

	commonerrors "ecommerce-be/common/error"
	"ecommerce-be/product/utils"
)

// Category Errors

var (
	// ErrCategoryExists is returned when a category with the same name already exists
	ErrCategoryExists = &commonerrors.AppError{
		Code:       utils.CATEGORY_EXISTS_CODE,
		Message:    utils.CATEGORY_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrCategoryNotFound is returned when a category is not found
	ErrCategoryNotFound = &commonerrors.AppError{
		Code:       utils.CATEGORY_NOT_FOUND_CODE,
		Message:    utils.CATEGORY_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrCategoryHasProducts is returned when trying to delete a category with products
	ErrCategoryHasProducts = &commonerrors.AppError{
		Code:       utils.CATEGORY_HAS_PRODUCTS_CODE,
		Message:    utils.CATEGORY_HAS_PRODUCTS_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrCategoryHasChildren is returned when trying to delete a category with child categories
	ErrCategoryHasChildren = &commonerrors.AppError{
		Code:       utils.CATEGORY_HAS_CHILDREN_CODE,
		Message:    utils.CATEGORY_HAS_CHILDREN_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrInvalidParentCategory is returned when parent category is invalid
	ErrInvalidParentCategory = &commonerrors.AppError{
		Code:       utils.INVALID_PARENT_CATEGORY_CODE,
		Message:    utils.INVALID_PARENT_CATEGORY_MSG,
		StatusCode: http.StatusBadRequest,
	}
)
