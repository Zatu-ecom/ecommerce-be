package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAttributeBasics tests basic attribute functionality
func TestAttributeBasics(t *testing.T) {
	t.Log("🧪 Testing basic attribute functionality...")

	t.Run("Test Data Structure", func(t *testing.T) {
		// Test that test data is properly structured
		assert.Len(t, TestAttributeData, 4, "Should have 4 test attributes")

		// Verify first attribute
		firstAttr := TestAttributeData[0]
		assert.Equal(t, "color", firstAttr.Key)
		assert.Equal(t, "Color", firstAttr.Name)
		assert.Equal(t, "string", firstAttr.DataType)
		assert.Len(t, firstAttr.AllowedValues, 4)
	})

	t.Run("Test Builder Functions", func(t *testing.T) {
		// Test attribute create request builder
		createReq := BuildAttributeCreateRequest("test_key", "Test Attribute", "string", "unit", "Test Description", []string{"value1", "value2"})
		assert.Equal(t, "test_key", createReq.Key)
		assert.Equal(t, "Test Attribute", createReq.Name)
		assert.Equal(t, "string", createReq.DataType)
		assert.Equal(t, "unit", createReq.Unit)
		assert.Equal(t, "Test Description", createReq.Description)
		assert.Len(t, createReq.AllowedValues, 2)

		// Test attribute update request builder
		updateReq := BuildAttributeUpdateRequest("updated_key", "Updated Attribute", "number", "kg", "Updated Description", []string{"value3"})
		assert.Equal(t, "Updated Attribute", updateReq.Name)
		assert.Equal(t, "number", updateReq.DataType)
		assert.Equal(t, "kg", updateReq.Unit)
		assert.Equal(t, "Updated Description", updateReq.Description)
		assert.Len(t, updateReq.AllowedValues, 1)
	})

	t.Log("✅ Basic attribute tests completed")
}

// TestAttributeValidation tests attribute validation rules
func TestAttributeValidation(t *testing.T) {
	t.Log("🧪 Testing attribute validation rules...")

	t.Run("Required Fields", func(t *testing.T) {
		// Test that required fields are properly handled
		createReq := BuildAttributeCreateRequest("test_key", "Test Name", "string", "", "Test Description", nil)
		// Verify that the request is properly built
		assert.NotEmpty(t, createReq.Key, "Key should be set")
		assert.NotEmpty(t, createReq.Name, "Name should be set")
		assert.NotEmpty(t, createReq.DataType, "DataType should be set")
	})

	t.Run("Data Type Validation", func(t *testing.T) {
		// Test that data types are properly validated
		validTypes := []string{"string", "number", "boolean", "array"}
		for _, dataType := range validTypes {
			createReq := BuildAttributeCreateRequest("test", "Test", dataType, "", "", nil)
			assert.Equal(t, dataType, createReq.DataType, "Data type should be %s", dataType)
		}
	})

	t.Log("✅ Attribute validation tests completed")
}

// TestAttributeBusinessRules tests business logic for attributes
func TestAttributeBusinessRules(t *testing.T) {
	t.Log("🧪 Testing attribute business rules...")

	t.Run("Allowed Values", func(t *testing.T) {
		// Test that allowed values are properly handled
		allowedValues := []string{"Red", "Blue", "Green"}
		createReq := BuildAttributeCreateRequest("color", "Color", "string", "", "Color attribute", allowedValues)

		assert.Len(t, createReq.AllowedValues, 3)
		assert.Contains(t, createReq.AllowedValues, "Red")
		assert.Contains(t, createReq.AllowedValues, "Blue")
		assert.Contains(t, createReq.AllowedValues, "Green")
	})

	t.Run("Unit Handling", func(t *testing.T) {
		// Test that units are properly handled
		createReq := BuildAttributeCreateRequest("weight", "Weight", "number", "kg", "Weight in kilograms", nil)
		assert.Equal(t, "kg", createReq.Unit)
	})

	t.Log("✅ Attribute business rules tests completed")
}
