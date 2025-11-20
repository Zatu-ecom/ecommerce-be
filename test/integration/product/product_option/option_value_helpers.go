package product_option

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// assertOptionValueFields verifies option value response has all required fields
func assertOptionValueFields(t *testing.T, valueData map[string]interface{}, expectedValue, expectedDisplayName string) {
	assert.NotNil(t, valueData["id"], "Option value should have id")
	assert.Equal(t, expectedValue, valueData["value"], "Option value mismatch")
	assert.Equal(t, expectedDisplayName, valueData["displayName"], "Display name mismatch")
	assert.NotNil(t, valueData["position"], "Option value should have position")
}

// assertOptionValueFieldsWithColor verifies option value with color code
func assertOptionValueFieldsWithColor(t *testing.T, valueData map[string]interface{}, expectedValue, expectedDisplayName, expectedColorCode string, expectedPosition int) {
	assertOptionValueFields(t, valueData, expectedValue, expectedDisplayName)
	assert.Equal(t, expectedColorCode, valueData["colorCode"], "Color code mismatch")
	assert.Equal(t, float64(expectedPosition), valueData["position"], "Position mismatch")
}
