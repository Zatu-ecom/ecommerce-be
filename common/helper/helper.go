package helper

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ToFormattedJSON converts an object to pretty-printed JSON (multi-line with indentation)
func ToFormattedJSON(obj interface{}) string {
	jsonBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return string(jsonBytes)
}

// ToJSON converts an object to compact JSON (single line)
func ToJSON(obj interface{}) string {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return string(jsonBytes)
}

// Primitive is a constraint that permits any primitive type
type Primitive interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 |
		~bool | ~string
}

// ParseCommaSeparated parses a comma-separated string pointer type into a slice of any primitive type
// Supports: integers, floats, bools, strings
//
// Examples:
//   - "1,2,3" -> []uint{1, 2, 3}
//   - "10,20,30" -> []int{10, 20, 30}
//   - "1.5,2.7,3.9" -> []float64{1.5, 2.7, 3.9}
//   - "true,false,true" -> []bool{true, false, true}
//   - "apple,banana,orange" -> []string{"apple", "banana", "orange"}
//
// Handles spaces and empty values gracefully
func ParseCommaSeparatedPtr[T Primitive](value *string) []T {
	if value == nil {
		return []T{}
	}
	return ParseCommaSeparated[T](*value)
}

// ParseCommaSeparated parses a comma-separated string into a slice of any primitive type
// Supports: integers, floats, bools, strings
//
// Examples:
//   - "1,2,3" -> []uint{1, 2, 3}
//   - "10,20,30" -> []int{10, 20, 30}
//   - "1.5,2.7,3.9" -> []float64{1.5, 2.7, 3.9}
//   - "true,false,true" -> []bool{true, false, true}
//   - "apple,banana,orange" -> []string{"apple", "banana", "orange"}
//
// Handles spaces and empty values gracefully
func ParseCommaSeparated[T Primitive](value string) []T {
	if value == "" {
		return []T{}
	}

	parts := strings.Split(value, ",")
	result := make([]T, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var parsed T
		var zero T

		// Use type switching based on the zero value's type
		switch any(zero).(type) {
		case string:
			// For strings, just use the trimmed part directly
			result = append(result, any(part).(T))
		case bool:
			// For bools, parse "true"/"false" (case-insensitive)
			if b, err := parseBool(part); err == nil {
				result = append(result, any(b).(T))
			}
		case float32, float64:
			// For floats, use %f format
			if _, err := fmt.Sscanf(part, "%f", &parsed); err == nil {
				result = append(result, parsed)
			}
		default:
			// For integers, use %d format
			if _, err := fmt.Sscanf(part, "%d", &parsed); err == nil {
				result = append(result, parsed)
			}
		}
	}

	return result
}

// parseBool is a helper to parse boolean values (case-insensitive)
func parseBool(s string) (bool, error) {
	lower := strings.ToLower(s)
	switch lower {
	case "true", "1", "yes", "y", "t":
		return true, nil
	case "false", "0", "no", "n", "f":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", s)
	}
}

// JoinToCommaSeparated converts a slice of any primitive type to a comma-separated string
// Supports: integers, floats, bools, strings
//
// Examples:
//   - []uint{1, 2, 3} -> "1,2,3"
//   - []int{10, 20, 30} -> "10,20,30"
//   - []float64{1.5, 2.7, 3.9} -> "1.5,2.7,3.9"
//   - []bool{true, false, true} -> "true,false,true"
//   - []string{"apple", "banana"} -> "apple,banana"
func JoinToCommaSeparated[T Primitive](values []T) string {
	if len(values) == 0 {
		return ""
	}

	parts := make([]string, len(values))

	for i, v := range values {
		// Use fmt.Sprintf with %v for universal formatting
		parts[i] = fmt.Sprintf("%v", v)
	}

	return strings.Join(parts, ",")
}

// ============================================================================
// Pointer Helpers
// ============================================================================

// StringPtr returns a pointer to a string
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to an int
func IntPtr(i int) *int {
	return &i
}

// UintPtr returns a pointer to a uint
func UintPtr(u uint) *uint {
	return &u
}

// BoolPtr returns a pointer to a bool
func BoolPtr(b bool) *bool {
	return &b
}
