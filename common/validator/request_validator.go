package validator

import (
	"reflect"

	commonError "ecommerce-be/common/error"
)

// RequireAtLeastOneField checks if a struct has at least one non-zero/non-nil field
// Works with any struct type containing pointers, strings, ints, bools, etc.
func RequireAtLeastOneField(s interface{}) error {
	v := reflect.ValueOf(s)
	
	// Handle pointer to struct
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return commonError.ErrInvalidRequestStruct.WithMessage("request cannot be nil")
		}
		v = v.Elem()
	}
	
	// Ensure it's a struct
	if v.Kind() != reflect.Struct {
		return commonError.ErrInvalidRequestStruct
	}
	
	// Check each field
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		
		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}
		
		// Check if field is non-zero
		if !isZeroValue(field) {
			return nil // Found at least one non-zero field
		}
	}
	
	return commonError.ErrNoFieldsProvided
}

// isZeroValue checks if a reflect.Value is zero/nil/empty
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map, reflect.Chan:
		return v.IsNil() || v.Len() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Struct:
		// For structs, check if it's the zero value
		return v.IsZero()
	default:
		return false
	}
}

// RequireAtLeastOneNonNilPointer checks if a struct has at least one non-nil pointer field
// Specifically for update requests where all fields are pointers
func RequireAtLeastOneNonNilPointer(s interface{}) error {
	v := reflect.ValueOf(s)
	
	// Handle pointer to struct
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return commonError.ErrInvalidRequestStruct.WithMessage("request cannot be nil")
		}
		v = v.Elem()
	}
	
	// Ensure it's a struct
	if v.Kind() != reflect.Struct {
		return commonError.ErrInvalidRequestStruct
	}
	
	// Check each pointer field
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		
		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}
		
		// Check if field is a pointer and non-nil
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			return nil
		}
	}
	
	return commonError.ErrNoFieldsProvided
}

// RequireAtLeastOneWithTag checks if a struct has at least one non-zero field with a specific tag
// Example: RequireAtLeastOneWithTag(req, "updateable", "true")
func RequireAtLeastOneWithTag(s interface{}, tagKey, tagValue string) error {
	v := reflect.ValueOf(s)
	
	// Handle pointer to struct
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return commonError.ErrInvalidRequestStruct.WithMessage("request cannot be nil")
		}
		v = v.Elem()
	}
	
	// Ensure it's a struct
	if v.Kind() != reflect.Struct {
		return commonError.ErrInvalidRequestStruct
	}
	
	t := v.Type()
	hasTaggedField := false
	
	// Check each field
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}
		
		// Check if field has the specified tag
		if tag := fieldType.Tag.Get(tagKey); tag == tagValue {
			hasTaggedField = true
			
			// Check if this tagged field is non-zero
			if !isZeroValue(field) {
				return nil
			}
		}
	}
	
	if !hasTaggedField {
		return commonError.ErrInvalidRequestStruct.WithMessage("no fields with specified tag found")
	}
	
	return commonError.ErrNoFieldsProvided
}
