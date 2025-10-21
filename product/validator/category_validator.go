package validator

import (
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

	parentCategory, err := v.categoryRepo.FindByID(*parentID)
	if err != nil {
		return err
	}
	if parentCategory == nil {
		return prodErrors.ErrInvalidParentCategory
	}

	return nil
}

// ValidateCircularReference validates that a category is not its own parent
func (v *CategoryValidator) ValidateCircularReference(categoryID uint, parentID *uint) error {
	if parentID == nil || *parentID == 0 {
		return nil
	}

	if *parentID == categoryID {
		return prodErrors.ErrInvalidParentCategory.WithMessage("Category cannot be its own parent")
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
func (v *CategoryValidator) ValidateNameChange(currentName, newName string, parentID *uint, categoryID uint) error {
	if currentName == newName {
		return nil // Name hasn't changed
	}

	return v.ValidateUniqueName(newName, parentID, &categoryID)
}
