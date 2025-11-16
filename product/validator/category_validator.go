package validator

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
)

// ValidateParentCategory validates that the parent category exists
// parentCategory should be nil if parentID is nil/0, otherwise should be the fetched parent category
func ValidateParentCategory(parentID *uint, parentCategory *entity.Category) error {
	if parentID == nil || *parentID == 0 {
		return nil // No parent is valid
	}

	if parentCategory == nil {
		return prodErrors.ErrInvalidParentCategory
	}

	return nil
}

// ValidateCircularReference validates that a category is not its own parent
// and that setting this parent won't create a circular reference in the hierarchy
// parentChain is the list of parent categories in the hierarchy (from immediate parent up to root)
func ValidateCircularReference(
	categoryID uint,
	parentID *uint,
	parentChain []*entity.Category,
) error {
	if parentID == nil || *parentID == 0 {
		return nil
	}

	// Check if trying to set itself as parent
	if *parentID == categoryID {
		return prodErrors.ErrInvalidParentCategory.WithMessage("Category cannot be its own parent")
	}

	// Check if the category appears in its parent chain
	// This prevents circular references like A->B->C->A
	for _, parent := range parentChain {
		if parent.ID == categoryID {
			return prodErrors.ErrInvalidParentCategory.WithMessage(
				"Cannot create circular reference in category hierarchy",
			)
		}
	}

	return nil
}

// ValidateUniqueName validates that the category name is unique within the same parent
// existingCategory should be nil if no category with this name+parent exists, otherwise the existing category
func ValidateUniqueName(
	name string,
	parentID *uint,
	excludeID *uint,
	existingCategory *entity.Category,
) error {
	if existingCategory != nil {
		// If excludeID is provided, allow the same name for the category being updated
		if excludeID == nil || existingCategory.ID != *excludeID {
			return prodErrors.ErrCategoryExists
		}
	}

	return nil
}

// ValidateCanDelete validates that a category can be deleted
// hasProducts and hasChildren should be provided from the service layer
func ValidateCanDelete(hasProducts bool, hasChildren bool) error {
	// Check if category has active products
	if hasProducts {
		return prodErrors.ErrCategoryHasProducts
	}

	// Check if category has active child categories
	if hasChildren {
		return prodErrors.ErrCategoryHasChildren
	}

	return nil
}

// ValidateNameChange validates that the name change is allowed
// existingCategory should be the category with the new name+parent (if it exists)
func ValidateNameChange(
	currentName, newName string,
	parentID *uint,
	categoryID uint,
	existingCategory *entity.Category,
) error {
	if currentName == newName {
		return nil // Name hasn't changed
	}

	return ValidateUniqueName(newName, parentID, &categoryID, existingCategory)
}

func ValidateCategoryOwnershipOrAdminAccess(
	roleLevel uint,
	sellerId uint,
	category *entity.Category,
) error {
	// Admin can delete any category (global or seller-specific)
	if roleLevel <= constants.ADMIN_ROLE_LEVEL {
		return nil
	}

	// Sellers can only delete their own seller-specific categories
	// They cannot delete global categories
	if category.IsGlobal {
		return prodErrors.ErrUnauthorizedCategoryUpdate
	}

	// Check if seller owns this category
	if category.SellerID == nil || *category.SellerID != sellerId {
		return prodErrors.ErrUnauthorizedCategoryUpdate
	}

	return nil
}
