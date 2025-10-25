package validator

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/repositories"
)

// CategoryValidator handles validation logic for categories
type CategoryValidator struct {
	categoryRepo repositories.CategoryRepository
}

// NewCategoryValidator creates a new instance of CategoryValidator
func NewCategoryValidator(categoryRepo repositories.CategoryRepository) *CategoryValidator {
	return &CategoryValidator{
		categoryRepo: categoryRepo,
	}
}

// ValidateParentCategory validates that the parent category exists
func (v *CategoryValidator) ValidateParentCategory(parentID *uint) error {
	if parentID == nil || *parentID == 0 {
		return nil // No parent is valid
	}

	_, err := v.categoryRepo.FindByID(*parentID)
	if err != nil {
		return prodErrors.ErrInvalidParentCategory
	}

	return nil
}

// ValidateCircularReference validates that a category is not its own parent
// and that setting this parent won't create a circular reference in the hierarchy
func (v *CategoryValidator) ValidateCircularReference(categoryID uint, parentID *uint) error {
	if parentID == nil || *parentID == 0 {
		return nil
	}

	// Check if trying to set itself as parent
	if *parentID == categoryID {
		return prodErrors.ErrInvalidParentCategory.WithMessage("Category cannot be its own parent")
	}

	// Check if the new parent is a descendant of this category
	// This prevents circular references like A->B->C->A
	currentParentID := parentID
	visited := make(map[uint]bool)

	for currentParentID != nil && *currentParentID != 0 {
		// Prevent infinite loops in case of existing circular references
		if visited[*currentParentID] {
			break
		}
		visited[*currentParentID] = true

		// If we find the category in its own parent chain, it's circular
		if *currentParentID == categoryID {
			return prodErrors.ErrInvalidParentCategory.WithMessage(
				"Cannot create circular reference in category hierarchy",
			)
		}

		// Get the parent category
		parentCategory, err := v.categoryRepo.FindByID(*currentParentID)
		if err != nil {
			// If parent not found, break the loop
			break
		}

		// Move up the chain
		currentParentID = parentCategory.ParentID
	}

	return nil
}

// ValidateUniqueName validates that the category name is unique within the same parent
func (v *CategoryValidator) ValidateUniqueName(name string, parentID *uint, excludeID *uint) error {
	existingCategory, err := v.categoryRepo.FindByNameAndParent(name, parentID)
	if err != nil {
		return err
	}

	if existingCategory != nil {
		// If excludeID is provided, allow the same name for the category being updated
		if excludeID == nil || existingCategory.ID != *excludeID {
			return prodErrors.ErrCategoryExists
		}
	}

	return nil
}

// ValidateCanDelete validates that a category can be deleted
func (v *CategoryValidator) ValidateCanDelete(id uint) error {
	// Check if category has active products
	hasProducts, err := v.categoryRepo.CheckHasProducts(id)
	if err != nil {
		return err
	}
	if hasProducts {
		return prodErrors.ErrCategoryHasProducts
	}

	// Check if category has active child categories
	hasChildren, err := v.categoryRepo.CheckHasChildren(id)
	if err != nil {
		return err
	}
	if hasChildren {
		return prodErrors.ErrCategoryHasChildren
	}

	return nil
}

// ValidateNameChange validates that the name change is allowed
func (v *CategoryValidator) ValidateNameChange(
	currentName, newName string,
	parentID *uint,
	categoryID uint,
) error {
	if currentName == newName {
		return nil // Name hasn't changed
	}

	return v.ValidateUniqueName(newName, parentID, &categoryID)
}

func (v *CategoryValidator) ValidateCategoryOwnershipOrAdminAccess(
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
