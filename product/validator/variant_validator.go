package validator

import (
	"errors"

	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"

	"gorm.io/gorm"
)

// VariantValidator handles validation logic for product variants
type VariantValidator struct {
	variantRepo repositories.VariantRepository
	productRepo repositories.ProductRepository
}

// NewVariantValidator creates a new instance of VariantValidator
func NewVariantValidator(
	variantRepo repositories.VariantRepository,
	productRepo repositories.ProductRepository,
) *VariantValidator {
	return &VariantValidator{
		variantRepo: variantRepo,
		productRepo: productRepo,
	}
}

// ValidateProductExists validates that a product exists
func (v *VariantValidator) ValidateProductExists(productID uint) error {
	_, err := v.productRepo.FindByID(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return prodErrors.ErrProductNotFound
		}
		return err
	}
	return nil
}

// ValidateVariantBelongsToProduct validates that a variant belongs to a product
func (v *VariantValidator) ValidateVariantBelongsToProduct(productID uint, variantID uint) error {
	variant, err := v.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return prodErrors.ErrVariantNotFound
		}
		return err
	}

	if variant.ProductID != productID {
		return prodErrors.ErrVariantNotFound // Variant doesn't belong to this product
	}

	return nil
}

// ValidateVariantOptions validates the options map structure
func (v *VariantValidator) ValidateVariantOptions(options map[string]string) error {
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
func (v *VariantValidator) ValidateVariantCombinationUnique(
	productID uint,
	optionsMap map[string]string,
) error {
	existingVariant, _ := v.variantRepo.FindVariantByOptions(productID, optionsMap)
	if existingVariant != nil {
		return prodErrors.ErrVariantCombinationExists
	}
	return nil
}

// ValidateCanDeleteVariant validates that a variant can be deleted (not the last one)
func (v *VariantValidator) ValidateCanDeleteVariant(productID uint) error {
	count, err := v.variantRepo.CountVariantsByProductID(productID)
	if err != nil {
		return err
	}

	if count <= 1 {
		return prodErrors.ErrLastVariantDeleteNotAllowed
	}

	return nil
}

// ValidateStockOperation validates the stock operation type and value
func (v *VariantValidator) ValidateStockOperation(
	request *model.UpdateVariantStockRequest,
	currentStock int,
) error {
	switch request.Operation {
	case "set", "add":
		// Always valid
		return nil
	case "subtract":
		if currentStock < request.Stock {
			return prodErrors.ErrInsufficientStockForOperation
		}
		return nil
	default:
		return prodErrors.ErrInvalidStockOperation
	}
}

// ValidateBulkUpdateRequest validates the bulk update request
func (v *VariantValidator) ValidateBulkUpdateRequest(
	request *model.BulkUpdateVariantsRequest,
) error {
	if len(request.Variants) == 0 {
		return prodErrors.ErrBulkUpdateEmptyList
	}
	return nil
}

// ValidateBulkVariantsExist validates that all variants in bulk update exist and belong to product
func (v *VariantValidator) ValidateBulkVariantsExist(productID uint, variantIDs []uint) error {
	existingVariants, err := v.variantRepo.FindVariantsByIDs(variantIDs)
	if err != nil {
		return err
	}

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

// ValidateOptionExists validates that a product option exists
func (v *VariantValidator) ValidateOptionExists(productID uint, optionName string) (*uint, error) {
	option, err := v.variantRepo.GetProductOptionByName(productID, optionName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrProductOptionNotFound
		}
		return nil, err
	}
	return &option.ID, nil
}

// ValidateOptionValueExists validates that an option value exists
func (v *VariantValidator) ValidateOptionValueExists(optionID uint, value string) (*uint, error) {
	optionValue, err := v.variantRepo.GetProductOptionValueByValue(optionID, value)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrProductOptionValueNotFound
		}
		return nil, err
	}
	return &optionValue.ID, nil
}
