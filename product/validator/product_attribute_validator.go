package validator

import (
	commonError "ecommerce-be/common/error"
	prodErrors "ecommerce-be/product/errors"
)

// ValidateProductAttributeValue validates the attribute value
func ValidateProductAttributeValue(
	value string,
	allowedValues []string,
) error {
	if value == "" {
		return commonError.ErrValidation.WithMessage("Attribute value cannot be empty")
	}

	// If allowed values are specified, validate against them
	if len(allowedValues) > 0 {
		for _, allowed := range allowedValues {
			if value == allowed {
				return nil
			}
		}
		return prodErrors.ErrInvalidAttributeValue
	}

	return nil
}

// ValidateProductAttributeAddRequest validates the add product attribute request
func ValidateProductAttributeAddRequest(
	attributeDefinitionID uint,
	value string,
	allowedValues []string,
) error {
	if attributeDefinitionID == 0 {
		return commonError.ErrValidation.WithMessage("Attribute definition ID is required")
	}

	return ValidateProductAttributeValue(value, allowedValues)
}

// ValidateProductAttributeUpdateRequest validates the update product attribute request
func ValidateProductAttributeUpdateRequest(
	value string,
	allowedValues []string,
) error {
	return ValidateProductAttributeValue(value, allowedValues)
}
