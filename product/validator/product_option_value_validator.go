package validator

import (
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/utils"
)

// ValidateSellerProductAndOption validates both product and option exist and belong together
// product and option should be fetched entities from the service layer
func ValidateSellerProductAndOption(
	sellerID uint,
	productID uint,
	product *entity.Product,
	option *entity.ProductOption,
) error {
	// Validate product exists
	if product == nil {
		return prodErrors.ErrProductNotFound
	}

	if sellerID != 0 && product.SellerID != sellerID {
		return prodErrors.ErrUnauthorizedProductAccess
	}

	// Validate option exists and belongs to product
	if option == nil {
		return prodErrors.ErrProductOptionNotFound
	}

	if option.ProductID != productID {
		return prodErrors.ErrProductOptionMismatch
	}

	return nil
}

// ValidateProductOptionValueUniqueness validates that option value is unique for an option
// existingValues should be the list of existing values for the option
func ValidateProductOptionValueUniqueness(
	value string,
	existingValues []entity.ProductOptionValue,
) error {
	normalizedValue := utils.ToLowerTrimmed(value)

	for _, val := range existingValues {
		if val.Value == normalizedValue {
			return prodErrors.ErrProductOptionValueExists
		}
	}
	return nil
}

// ValidateProductOptionValueBelongsToOption validates that a value belongs to an option
// optionValue should be the fetched option value entity
func ValidateProductOptionValueBelongsToOption(
	optionID uint,
	optionValue *entity.ProductOptionValue,
) error {
	if optionValue == nil {
		return prodErrors.ErrProductOptionValueNotFound
	}

	if optionValue.OptionID != optionID {
		return prodErrors.ErrProductOptionValueMismatch
	}

	return nil
}

// ValidateProductOptionValueNotInUse validates that a value is not being used by any variants
// inUse indicates if the value is being used by variants
// variantCount is the number of variants using this value
func ValidateProductOptionValueNotInUse(inUse bool, variantCount int) error {
	if inUse {
		return prodErrors.ErrProductOptionValueInUse.WithMessagef(
			"%s (used by %d variants)",
			utils.PRODUCT_OPTION_VALUE_IN_USE_MSG,
			variantCount,
		)
	}

	return nil
}

// ValidateBulkProductOptionValuesUniqueness validates bulk option values for duplicates
// existingValues should be the list of existing values for the option
func ValidateBulkProductOptionValuesUniqueness(
	values []string,
	existingValues []entity.ProductOptionValue,
) error {
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
