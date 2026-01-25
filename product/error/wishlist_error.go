package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/utils"
)

// Wishlist Errors

var (
	// ErrWishlistNotFound is returned when a wishlist is not found
	ErrWishlistNotFound = &commonError.AppError{
		Code:       utils.WISHLIST_NOT_FOUND_CODE,
		Message:    utils.WISHLIST_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrWishlistNameExists is returned when wishlist name already exists for user
	ErrWishlistNameExists = &commonError.AppError{
		Code:       utils.WISHLIST_NAME_EXISTS_CODE,
		Message:    utils.WISHLIST_NAME_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrMaxWishlistsReached is returned when user has max wishlists
	ErrMaxWishlistsReached = &commonError.AppError{
		Code:       utils.MAX_WISHLISTS_REACHED_CODE,
		Message:    utils.MAX_WISHLISTS_REACHED_MSG,
		StatusCode: http.StatusUnprocessableEntity,
	}

	// ErrUnauthorizedWishlist is returned when user doesn't own the wishlist
	ErrUnauthorizedWishlist = &commonError.AppError{
		Code:       utils.UNAUTHORIZED_WISHLIST_CODE,
		Message:    utils.UNAUTHORIZED_WISHLIST_MSG,
		StatusCode: http.StatusForbidden,
	}

	// ErrCannotDeleteDefault is returned when trying to delete the default wishlist
	ErrCannotDeleteDefault = &commonError.AppError{
		Code:       utils.CANNOT_DELETE_DEFAULT_CODE,
		Message:    utils.CANNOT_DELETE_DEFAULT_MSG,
		StatusCode: http.StatusUnprocessableEntity,
	}
)

// Wishlist Item Errors

var (
	// ErrWishlistItemNotFound is returned when a wishlist item is not found
	ErrWishlistItemNotFound = &commonError.AppError{
		Code:       utils.WISHLIST_ITEM_NOT_FOUND_CODE,
		Message:    utils.WISHLIST_ITEM_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrWishlistItemExists is returned when item already exists in wishlist
	ErrWishlistItemExists = &commonError.AppError{
		Code:       utils.WISHLIST_ITEM_EXISTS_CODE,
		Message:    utils.WISHLIST_ITEM_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrUnauthorizedWishlistItem is returned when user doesn't own the wishlist item
	ErrUnauthorizedWishlistItem = &commonError.AppError{
		Code:       utils.UNAUTHORIZED_WISHLIST_ITEM_CODE,
		Message:    utils.UNAUTHORIZED_WISHLIST_ITEM_MSG,
		StatusCode: http.StatusForbidden,
	}

	// ErrSameWishlistMove is returned when trying to move item to the same wishlist
	ErrSameWishlistMove = &commonError.AppError{
		Code:       utils.SAME_WISHLIST_MOVE_CODE,
		Message:    utils.SAME_WISHLIST_MOVE_MSG,
		StatusCode: http.StatusBadRequest,
	}
)
