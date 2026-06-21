package validator

import (
	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/error"
	"ecommerce-be/product/model"
)

// ValidateProductCategoryExists checks if a category exists
// category should be the fetched category entity
func ValidateProductCategoryExists(category *entity.Category) error {
	if category == nil {
		return prodErrors.ErrInvalidCategory
	}
	return nil
}

// ValidateProductExistsAndOwnership checks if a product exists and seller has access
// Reuses ValidateProductBelongsToSeller for consistency
func ValidateProductExistsAndOwnership(product *entity.Product, sellerID *uint) error {
	if product == nil {
		return prodErrors.ErrProductNotFound
	}

	if sellerID != nil {
		// Reuse existing validation logic
		return ValidateProductBelongsToSeller(product, *sellerID)
	}

	return nil
}

// ValidateProductVariantRequirements validates variant-related requirements for product creation.
// Simple products may omit variants when top-level price is provided and no options are defined.
func ValidateProductVariantRequirements(req model.ProductCreateRequest) error {
	if len(req.Variants) > 0 {
		return nil
	}

	if len(req.Options) > 0 {
		return commonError.ErrValidation.WithMessage(
			"variants are required when product options are provided",
		)
	}

	if req.Price <= 0 {
		return commonError.ErrValidation.WithMessage(
			"price is required when variants are not provided",
		)
	}

	return nil
}

// ValidateProductVariantSKUsUnique checks that all variant SKUs in the request are unique
func ValidateProductVariantSKUsUnique(variants []model.CreateVariantRequest) error {
	skuMap := make(map[string]bool)
	for _, variant := range variants {
		if skuMap[variant.SKU] {
			return commonError.ErrValidation.WithMessagef("duplicate variant SKU: %s", variant.SKU)
		}
		skuMap[variant.SKU] = true
	}
	return nil
}

// ValidateProductCreateRequest validates the entire product creation request
// category should be the fetched category entity
func ValidateProductCreateRequest(req model.ProductCreateRequest, category *entity.Category) error {
	// Validate category exists
	if err := ValidateProductCategoryExists(category); err != nil {
		return err
	}

	// Validate variant requirements
	if err := ValidateProductVariantRequirements(req); err != nil {
		return err
	}

	// Validate variant SKUs are unique if manual variants provided
	if len(req.Variants) > 0 {
		if err := ValidateProductVariantSKUsUnique(req.Variants); err != nil {
			return err
		}
	}

	return nil
}

// ValidateProductUpdateRequest validates product update request
// product should be the fetched product entity
// category should be the fetched category entity if being updated
func ValidateProductUpdateRequest(
	product *entity.Product,
	sellerID *uint,
	req model.ProductUpdateRequest,
	category *entity.Category,
) error {
	// Verify product exists and check ownership
	if err := ValidateProductExistsAndOwnership(product, sellerID); err != nil {
		return err
	}

	// Validate category if being updated (pointer not nil)
	if req.CategoryID != nil && *req.CategoryID != 0 {
		if err := ValidateProductCategoryExists(category); err != nil {
			return err
		}
	}

	if req.Price != nil && *req.Price <= 0 {
		return commonError.ErrValidation.WithMessage("price must be greater than 0")
	}

	return nil
}
