package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestBasicSetup verifies that our test environment can be set up without database dependencies
func TestBasicSetup(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test router
	router := gin.New()

	// Add a simple test endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"message": "Test environment is working",
		})
	})

	// Create a test request
	req, err := http.NewRequest(http.MethodGet, "/health", nil)
	assert.NoError(t, err)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Assert the response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "Test environment is working", response["message"])
}

// TestJSONRequest verifies that our test helpers can handle JSON requests
func TestJSONRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add a test endpoint that accepts JSON
	router.POST("/test-json", func(c *gin.Context) {
		var requestData map[string]interface{}
		if err := c.ShouldBindJSON(&requestData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"received": requestData,
			"message":  "JSON processed successfully",
		})
	})

	// Create test data
	testData := map[string]interface{}{
		"name":  "test",
		"value": 123,
	}

	jsonData, err := json.Marshal(testData)
	assert.NoError(t, err)

	// Create request
	req, err := http.NewRequest(http.MethodPost, "/test-json", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "JSON processed successfully", response["message"])
}

// TestConstants verifies that our test constants are accessible
func TestConstants(t *testing.T) {
	// Test that our constants from test_helpers.go are accessible
	assert.NotEmpty(t, TestEmail)
	assert.NotEmpty(t, TestPassword)
	assert.NotEmpty(t, TestFirstName)
	assert.NotEmpty(t, TestLastName)
	assert.NotEmpty(t, EndpointRegister)
	assert.NotEmpty(t, EndpointLogin)

	// Verify constant values
	assert.Equal(t, "john.doe@example.com", TestEmail)
	assert.Equal(t, "Password123!", TestPassword)
	assert.Equal(t, "John", TestFirstName)
	assert.Equal(t, "Doe", TestLastName)
}

// TestHelperFunctions verifies that our test helper functions work
func TestHelperFunctions(t *testing.T) {
	// Test that APIResponse struct is available
	response := APIResponse{
		Success: true,
		Message: "Test message",
		Data:    "Test data",
	}

	assert.True(t, response.Success)
	assert.Equal(t, "Test message", response.Message)
	assert.Equal(t, "Test data", response.Data)
}
