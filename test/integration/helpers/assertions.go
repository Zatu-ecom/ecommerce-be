package helpers

import (
	"fmt"
	"log"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// AssertSuccessResponse verifies a successful API response
func AssertSuccessResponse(
	t *testing.T,
	w *httptest.ResponseRecorder,
	expectedStatus int,
) map[string]interface{} {
	assert.Equal(
		t,
		expectedStatus,
		w.Code,
		"Expected status code %d but got %d",
		expectedStatus,
		w.Code,
	)

	response := ParseResponse(t, w.Body)
	assert.True(t, response["success"].(bool), "Response should be successful")

	return response
}

// AssertErrorResponse verifies an error API response
func AssertErrorResponse(
	t *testing.T,
	w *httptest.ResponseRecorder,
	expectedStatus int,
) map[string]interface{} {
	assert.Equal(
		t,
		expectedStatus,
		w.Code,
		"Expected status code %d but got %d",
		expectedStatus,
		w.Code,
	)

	response := ParseResponse(t, w.Body)
	log.Println("Error Response:", response)

	assert.False(t, response["success"].(bool), "Response should not be successful")

	return response
}

// GetResponseData extracts the data field from response
func GetResponseData(
	t *testing.T,
	response map[string]interface{},
	dataKey string,
) map[string]interface{} {
	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok, "Response should contain data field")

	result, ok := data[dataKey].(map[string]interface{})
	assert.True(t, ok, fmt.Sprintf("Data should contain %s field", dataKey))

	return result
}

// AssertCategoryFields verifies category response has all required fields
func AssertCategoryFields(t *testing.T, category map[string]interface{}, expectedName string) {
	assert.NotNil(t, category["id"], "Category should have id")
	assert.Equal(t, expectedName, category["name"], "Category name mismatch")
	assert.NotNil(t, category["createdAt"], "Category should have createdAt")
	assert.NotNil(t, category["updatedAt"], "Category should have updatedAt")
}

// AssertGlobalCategory verifies that a category is global (admin-created)
func AssertGlobalCategory(t *testing.T, category map[string]interface{}) {
	assert.True(t, category["isGlobal"].(bool), "Category should be global")
	assert.Nil(t, category["sellerId"], "Global category should not have sellerId")
}

// AssertSellerCategory verifies that a category is seller-specific
func AssertSellerCategory(t *testing.T, category map[string]interface{}, expectedSellerID uint) {
	assert.False(t, category["isGlobal"].(bool), "Category should not be global")
	assert.NotNil(t, category["sellerId"], "Seller category should have sellerId")
	assert.Equal(t, float64(expectedSellerID), category["sellerId"].(float64), "Seller ID mismatch")
}
