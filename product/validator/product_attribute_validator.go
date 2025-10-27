package validator

import (
	commonError "ecommerce-be/common/error"
	prodErrors "ecommerce-be/product/errors"
)

// ProductAttributeValidator handles validation logic for product attributes
type ProductAttributeValidator struct{}

// NewProductAttributeValidator creates a new instance of ProductAttributeValidator
func NewProductAttributeValidator() *ProductAttributeValidator {
	return &ProductAttributeValidator{}
}

// ValidateAttributeValue validates the attribute value
func (v *ProductAttributeValidator) ValidateAttributeValue(
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

// ValidateSortOrder validates the sort order
func (v *ProductAttributeValidator) ValidateSortOrder(sortOrder uint) error {
	// Sort order can be any non-negative value, including 0
	// No additional validation needed for now
	return nil
}

// ValidateAddRequest validates the add product attribute request
func (v *ProductAttributeValidator) ValidateAddRequest(
	attributeDefinitionID uint,
	value string,
	allowedValues []string,
) error {
	if attributeDefinitionID == 0 {
		return commonError.ErrValidation.WithMessage("Attribute definition ID is required")
	}

	return v.ValidateAttributeValue(value, allowedValues)
}

// ValidateUpdateRequest validates the update product attribute request
func (v *ProductAttributeValidator) ValidateUpdateRequest(
	value string,
	allowedValues []string,
) error {
	return v.ValidateAttributeValue(value, allowedValues)
}
