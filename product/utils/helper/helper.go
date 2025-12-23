package helper

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ecommerce-be/common/db"
)

// NormalizeToSnakeCase converts a string to lowercase snake_case format
// Replaces spaces with underscores and removes special characters
// Example: "Product Color" -> "product_color"
// Example: "User-Name!" -> "username"
func NormalizeToSnakeCase(str string) string {
	// Convert to lowercase
	str = strings.ToLower(str)
	// Replace spaces with underscores
	str = strings.ReplaceAll(str, " ", "_")
	// Remove special characters, keep only alphanumeric and underscores
	str = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, str)
	return str
}

// ToLowerTrimmed converts a string to lowercase and trims spaces
// Example:
//   - "  Red  " -> "red"
//   - "  BLUE " -> "blue"
func ToLowerTrimmed(str string) string {
	return strings.ToLower(strings.TrimSpace(str))
}

// GetDisplayNameOrDefault returns the display name if it exists, otherwise returns the default name
func GetDisplayNameOrDefault(displayName, defaultName string) string {
	if displayName != "" {
		return displayName
	}
	return defaultName
}

// ParseQueryParamsToMap parses query parameters into a map, excluding pagination and common filter params
// Example: color=red&size=m becomes map[string]string{"color": "red", "size": "m"}
// This is a generic function that can be used for any query parameters
func ParseQueryParamsToMap(
	queryParams map[string][]string,
	excludeParams []string,
) map[string]string {
	result := make(map[string]string)

	// Create a map of excluded parameters for quick lookup
	excludeMap := make(map[string]bool)
	for _, param := range excludeParams {
		excludeMap[param] = true
	}

	// Add all query parameters that are not in the exclude list
	for paramName, values := range queryParams {
		// Skip if in exclude list
		if excludeMap[paramName] {
			continue
		}

		// Take the first value if multiple values exist
		if len(values) > 0 && values[0] != "" {
			result[paramName] = values[0]
		}
	}

	return result
}

// ParseOptionsFromQuery parses option query parameters into a map
// This specifically excludes common pagination and filter parameters
// Example: color=red&size=m&page=1 becomes map[string]string{"color": "red", "size": "m"}
func ParseOptionsFromQuery(
	queryParams map[string][]string,
	defaultExcludes []string,
) map[string]string {
	// Common non-option query parameters to exclude
	excludeParams := []string{
		"page", "limit", "offset", // Pagination
		"sort", "sortBy", "sortOrder", "order", // Sorting
		"search", "query", "q", // Search
		"filter", "filters", // Generic filters
		"minPrice", "maxPrice", "inStock", "isPopular", "brand", // Product filters
		"categoryId", "category", // Category filters
	}

	excludeParams = append(excludeParams, defaultExcludes...)

	return ParseQueryParamsToMap(queryParams, excludeParams)
}

// ValidateVariantOptions validates that the provided options are valid
func ValidateVariantOptions(options map[string]string) error {
	if len(options) == 0 {
		return fmt.Errorf("at least one option must be provided")
	}

	for optionName, optionValue := range options {
		if optionName == "" {
			return fmt.Errorf("option name cannot be empty")
		}
		if optionValue == "" {
			return fmt.Errorf("option value for '%s' cannot be empty", optionName)
		}
	}

	return nil
}

// FormatTimestamp formats a time.Time to RFC3339 string
func FormatTimestamp(t time.Time) string {
	return t.Format(time.RFC3339)
}

// NewBaseEntity creates a new BaseEntity with current timestamp
func NewBaseEntity() db.BaseEntity {
	now := time.Now()
	return db.BaseEntity{
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// GetBoolOrDefault returns the value of the pointer if not nil, otherwise returns the default value
func GetBoolOrDefault(ptr *bool, defaultValue bool) bool {
	if ptr != nil {
		return *ptr
	}
	return defaultValue
}

// GetPositionOrDefault returns the requested position if not zero, otherwise returns the default position
func GetPositionOrDefault(requestedPosition, defaultPosition int) int {
	if requestedPosition == 0 {
		return defaultPosition
	}
	return requestedPosition
}

// GetOrEmptySlice returns the slice if not nil, otherwise returns an empty slice
func GetOrEmptySlice[T any](slice []T) []T {
	if slice != nil {
		return slice
	}
	return []T{}
}

func MapToPrettyJSON(data map[string]interface{}) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ") // 2-space indent
	if err != nil {
		return "", err
	}
	return string(b), nil
}