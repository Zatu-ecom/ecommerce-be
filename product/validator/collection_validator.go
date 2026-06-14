package validator

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/error"
)

// ValidateCollectionOwnershipOrAdminAccess ensures the user can mutate the collection
func ValidateCollectionOwnershipOrAdminAccess(
	roleLevel uint,
	sellerID uint,
	collection *entity.Collection,
) error {
	if roleLevel <= constants.ADMIN_ROLE_LEVEL {
		return nil
	}
	if collection.SellerID != sellerID {
		return prodErrors.ErrUnauthorizedCollectionAccess
	}
	return nil
}

// ValidateCollectionReadable ensures the collection is visible to the requester
func ValidateCollectionReadable(
	sellerIDPtr *uint,
	collection *entity.Collection,
) error {
	if sellerIDPtr == nil {
		return nil
	}
	if collection.SellerID != *sellerIDPtr {
		return prodErrors.ErrUnauthorizedCollectionAccess
	}
	return nil
}
