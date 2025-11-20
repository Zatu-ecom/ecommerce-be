package validator

import (
	"regexp"

	commonError "ecommerce-be/common/error"
	prodErrors "ecommerce-be/product/errors"
)

// ValidateKey validates the attribute key format
// Key must contain only lowercase letters, numbers, and underscores
func ValidateKey(key string) error {
	keyPattern := regexp.MustCompile(`^[a-z0-9_]+$`)
	if !keyPattern.MatchString(key) {
		return prodErrors.ErrInvalidAttributeKey
	}
	return nil
}

// ValidateAllowedValues validates the allowed values for an attribute
func ValidateAllowedValues(allowedValues []string) error {
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
