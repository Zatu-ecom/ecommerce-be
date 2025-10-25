package validator

import (
	"regexp"

	commonError "ecommerce-be/common/error"
	prodErrors "ecommerce-be/product/errors"
)

// AttributeValidator handles validation logic for attribute definitions
type AttributeValidator struct {
	keyPattern *regexp.Regexp
	validTypes []string
}

// NewAttributeValidator creates a new instance of AttributeValidator
func NewAttributeValidator() *AttributeValidator {
	return &AttributeValidator{
		keyPattern: regexp.MustCompile(`^[a-z0-9_]+$`),
		validTypes: []string{"string", "number", "boolean", "array"},
	}
}

// ValidateKey validates the attribute key format
// Key must contain only lowercase letters, numbers, and underscores
func (v *AttributeValidator) ValidateKey(key string) error {
	if !v.keyPattern.MatchString(key) {
		return prodErrors.ErrInvalidAttributeKey
	}
	return nil
}

// ValidateDataType validates the data type
func (v *AttributeValidator) ValidateDataType(dataType string) error {
	for _, validType := range v.validTypes {
		if dataType == validType {
			return nil
		}
	}
	return prodErrors.ErrInvalidDataType
}

// ValidateAllowedValues validates the allowed values for an attribute
func (v *AttributeValidator) ValidateAllowedValues(allowedValues []string) error {
	// Check for duplicate values
	seen := make(map[string]bool)
	for _, value := range allowedValues {
		if seen[value] {
			return commonError.ErrValidation.WithMessage("Duplicate values found in allowedValues")
		}
		seen[value] = true
	}
	return nil
}

// ValidateUpdateRequest validates that at least one field is being updated
func (v *AttributeValidator) ValidateUpdateRequest(
	name, unit string,
	allowedValues []string,
) error {
	if name == "" && unit == "" && len(allowedValues) == 0 {
		return commonError.ErrValidation.WithMessage(
			"At least one field must be provided for update",
		)
	}
	return nil
}
