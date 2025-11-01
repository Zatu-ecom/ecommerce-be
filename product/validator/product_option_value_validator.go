package validator

import (
	"errors"

	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/utils"

	"gorm.io/gorm"
)

// ProductOptionValueValidator handles validation logic for product option values
type ProductOptionValueValidator struct {
	optionRepo  repositories.ProductOptionRepository
	productRepo repositories.ProductRepository
}

// NewProductOptionValueValidator creates a new instance of ProductOptionValueValidator
func NewProductOptionValueValidator(
	optionRepo repositories.ProductOptionRepository,
	productRepo repositories.ProductRepository,
) *ProductOptionValueValidator {
	return &ProductOptionValueValidator{
		optionRepo:  optionRepo,
		productRepo: productRepo,
	}
}

// ValidateSellerProductAndOption validates both product and option exist and belong together
func (v *ProductOptionValueValidator) ValidateSellerProductAndOption(
	sellerID uint,
	productID uint,
	optionID uint,
) error {
	// Validate product exists
	product, err := v.productRepo.FindByID(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return prodErrors.ErrProductNotFound
		}
		return err
	}

	if sellerID != 0 && product.SellerID != sellerID {
		return prodErrors.ErrUnauthorizedProductAccess
	}

	// Validate option exists and belongs to product
	option, err := v.optionRepo.FindOptionByID(optionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return prodErrors.ErrProductOptionNotFound
		}
		return err
	}

	if option.ProductID != productID {
		return prodErrors.ErrProductOptionMismatch
	}

	return nil
}

// ValidateOptionValueUniqueness validates that option value is unique for an option
func (v *ProductOptionValueValidator) ValidateOptionValueUniqueness(
	optionID uint,
	value string,
) error {
	normalizedValue := utils.ToLowerTrimmed(value)

	existingValues, err := v.optionRepo.FindOptionValuesByOptionID(optionID)
	if err != nil {
		return err
	}

	for _, val := range existingValues {
		if val.Value == normalizedValue {
			return prodErrors.ErrProductOptionValueExists
		}
	}
	return nil
}

// ValidateOptionValueBelongsToOption validates that a value belongs to an option
func (v *ProductOptionValueValidator) ValidateOptionValueBelongsToOption(
	valueID uint,
	optionID uint,
) error {
	optionValue, err := v.optionRepo.FindOptionValueByID(valueID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return prodErrors.ErrProductOptionValueNotFound
		}
		return err
	}

	if optionValue.OptionID != optionID {
		return prodErrors.ErrProductOptionValueMismatch
	}

	return nil
}

// ValidateOptionValueNotInUse validates that a value is not being used by any variants
func (v *ProductOptionValueValidator) ValidateOptionValueNotInUse(valueID uint) error {
	inUse, variantIDs, err := v.optionRepo.CheckOptionValueInUse(valueID)
	if err != nil {
		return err
	}

	if inUse {
		return prodErrors.ErrProductOptionValueInUse.WithMessagef(
			"%s (used by %d variants)",
			utils.PRODUCT_OPTION_VALUE_IN_USE_MSG,
			len(variantIDs),
		)
	}

	return nil
}

// ValidateBulkOptionValuesUniqueness validates bulk option values for duplicates
func (v *ProductOptionValueValidator) ValidateBulkOptionValuesUniqueness(
	optionID uint,
	values []string,
) error {
	// Get existing values
	existingValues, err := v.optionRepo.FindOptionValuesByOptionID(optionID)
	if err != nil {
		return err
	}

	existingValueMap := make(map[string]bool)
	for _, val := range existingValues {
		existingValueMap[val.Value] = true
	}

	// Check for duplicates in existing and within batch
	valueSet := make(map[string]bool)
	for _, value := range values {
		normalizedValue := utils.ToLowerTrimmed(value)

		// Check against existing values
		if existingValueMap[normalizedValue] {
			return prodErrors.ErrProductOptionValueExists.WithMessagef(
				"%s: %s",
				utils.PRODUCT_OPTION_VALUE_EXISTS_MSG,
				normalizedValue,
			)
		}

		// Check for duplicates within batch
		if valueSet[normalizedValue] {
			return prodErrors.ErrProductOptionValueExists.WithMessagef(
				"%s: %s",
				utils.PRODUCT_OPTION_VALUE_DUPLICATE_IN_BATCH_MSG,
				normalizedValue,
			)
		}

		valueSet[normalizedValue] = true
	}

	return nil
}
