package validator

import (
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/model"
)

// ValidateVariantProductAndSeller validates that a product exists and seller has access
// Reuses ValidateProductBelongsToSeller for consistency
func ValidateVariantProductAndSeller(product *entity.Product, sellerID uint) error {
	// Variant creation with multiple options - admin check
	if sellerID == 0 {
		return nil
	}

	// Reuse the existing validation logic
	return ValidateProductBelongsToSeller(product, sellerID)
}

// ValidateVariantBelongsToProduct validates that a variant belongs to a product
// variant should be the fetched variant entity
func ValidateVariantBelongsToProduct(productID uint, variant *entity.ProductVariant) error {
	if variant == nil {
		return prodErrors.ErrVariantNotFound
	}

	if variant.ProductID != productID {
		return prodErrors.ErrVariantNotFound // Variant doesn't belong to this product
	}

	return nil
}

// ValidateVariantOptions validates the options map structure
func ValidateVariantOptions(options map[string]string) error {
	if len(options) == 0 {
		return prodErrors.ErrProductHasNoOptions
	}

	for optionName, value := range options {
		if optionName == "" {
			return prodErrors.ErrProductOptionNotFound
		}
		if value == "" {
			return prodErrors.ErrProductOptionValueNotFound
		}
	}

	return nil
}

// ValidateVariantCombinationUnique validates that variant combination doesn't already exist
// existingVariant should be nil if no variant with this combination exists
func ValidateVariantCombinationUnique(existingVariant *entity.ProductVariant) error {
	if existingVariant != nil {
		return prodErrors.ErrVariantCombinationExists
	}
	return nil
}

// ValidateCanDeleteVariant validates that a variant can be deleted (not the last one)
// variantCount is the total number of variants for the product
func ValidateCanDeleteVariant(variantCount int64) error {
	if variantCount <= 1 {
		return prodErrors.ErrLastVariantDeleteNotAllowed
	}

	return nil
}

// ValidateBulkVariantUpdateRequest validates the bulk update request
func ValidateBulkVariantUpdateRequest(request *model.BulkUpdateVariantsRequest) error {
	if len(request.Variants) == 0 {
		return prodErrors.ErrBulkUpdateEmptyList
	}
	return nil
}

// ValidateBulkVariantsExist validates that all variants in bulk update exist and belong to product
// existingVariants should be the fetched variants, variantIDs is the requested IDs
func ValidateBulkVariantsExist(productID uint, variantIDs []uint, existingVariants []entity.ProductVariant) error {
	// Validate count matches
	if len(existingVariants) != len(variantIDs) {
		return prodErrors.ErrBulkUpdateVariantNotFound
	}

	// Validate all belong to the product
	for _, variant := range existingVariants {
		if variant.ProductID != productID {
			return prodErrors.ErrBulkUpdateVariantNotFound
		}
	}

	return nil
}

// ValidateProductOptionExists validates that a product option exists
// option should be the fetched option entity, returns option ID if valid
func ValidateProductOptionExists(option *entity.ProductOption) (*uint, error) {
	if option == nil {
		return nil, prodErrors.ErrProductOptionNotFound
	}
	return &option.ID, nil
}

// ValidateProductOptionValueExists validates that an option value exists
// optionValue should be the fetched option value entity, returns value ID if valid
func ValidateProductOptionValueExists(optionValue *entity.ProductOptionValue) (*uint, error) {
	if optionValue == nil {
		return nil, prodErrors.ErrProductOptionValueNotFound
	}
	return &optionValue.ID, nil
}
