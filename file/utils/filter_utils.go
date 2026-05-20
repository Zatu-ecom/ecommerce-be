package utils

import "ecommerce-be/common/helper"

// ParseUintFilterList parses a comma-separated string into a slice of uint.
// Delegates to the shared generic helper in common/helper.
func ParseUintFilterList(raw string) []uint {
	return helper.ParseCommaSeparated[uint](raw)
}

// ParseStringFilterList parses a comma-separated string into a slice of strings.
// Delegates to the shared generic helper in common/helper.
func ParseStringFilterList(raw string) []string {
	return helper.ParseCommaSeparated[string](raw)
}
