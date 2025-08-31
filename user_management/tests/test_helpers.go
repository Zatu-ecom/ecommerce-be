package tests

import (
	"testing"
	"time"

	"ecommerce-be/user_management/entity"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Test constants
const (
	// Common test data
	TestPassword    = "Password123!"
	TestNewPassword = "NewPassword123!"
	TestPhone       = "+1234567890"
	TestDateOfBirth = "1990-01-01"
	TestGender      = "male"
	TestEmail       = "john.doe@example.com"
	TestEmailLogin  = "john.login@example.com"
	TestFirstName   = "John"
	TestLastName    = "Doe"
	TestCountry     = "USA"
	TestState       = "NY"
	TestCity        = "New York"
	TestZipCode     = "10001"
	TestStreet      = "123 Main St"

	// API endpoints
	EndpointRegister       = "/api/auth/register"
	EndpointLogin          = "/api/auth/login"
	EndpointRefresh        = "/api/auth/refresh"
	EndpointLogout         = "/api/auth/logout"
	EndpointProfile        = "/api/users/profile"
	EndpointUpdateProfile  = "/api/users/profile"
	EndpointAddresses      = "/api/users/addresses"
	EndpointUpdateAddress  = "/api/users/addresses/1"
	EndpointSetDefault     = "/api/users/addresses/1/default"
	EndpointChangePassword = "/api/users/password"
	EndpointHealth         = "/health"

	// Test server settings
	TestRedisURL      = "localhost:6379"
	TestRedisPassword = ""
	TestRedisDB       = 0
)

// APIResponse represents the standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
	Code    string      `json:"code,omitempty"`
}

// TestDataFactory provides methods to create test data
type TestDataFactory struct {
	db *gorm.DB
}

// NewTestDataFactory creates a new test data factory
func NewTestDataFactory(db *gorm.DB) *TestDataFactory {
	return &TestDataFactory{db: db}
}

// CreateTestUser creates a test user in the database
func (f *TestDataFactory) CreateTestUser(t *testing.T, email string) *entity.User {
	user := &entity.User{
		FirstName:   TestFirstName,
		LastName:    TestLastName,
		Email:       email,
		Password:    "$2a$10$N9qo8uLOickgx2ZMRZoMy.dBt/nHX85tF5.XjSe6R4u2k5t5zO5aS", // hashed "Password123!"
		Phone:       TestPhone,
		DateOfBirth: TestDateOfBirth,
		Gender:      TestGender,
		IsActive:    true,
	}

	err := f.db.Create(user).Error
	require.NoError(t, err)

	return user
}

// CreateTestAddress creates a test address for a user
func (f *TestDataFactory) CreateTestAddress(t *testing.T, userID uint, isDefault bool) *entity.Address {
	address := &entity.Address{
		UserID:    userID,
		Street:    TestStreet,
		City:      TestCity,
		State:     TestState,
		ZipCode:   TestZipCode,
		Country:   TestCountry,
		IsDefault: isDefault,
	}

	err := f.db.Create(address).Error
	require.NoError(t, err)

	return address
}

// CleanupTestData removes all test data from the database
func (f *TestDataFactory) CleanupTestData(t *testing.T) {
	// Delete in order due to foreign key constraints
	err := f.db.Where("1 = 1").Delete(&entity.Address{}).Error
	require.NoError(t, err)

	err = f.db.Where("1 = 1").Delete(&entity.User{}).Error
	require.NoError(t, err)
}

// WaitForAsync waits for asynchronous operations to complete
func WaitForAsync(duration time.Duration) {
	time.Sleep(duration)
}

// AssertResponseStructure validates the basic response structure
func AssertResponseStructure(t *testing.T, response APIResponse, expectSuccess bool) {
	if expectSuccess {
		require.True(t, response.Success, "Response should indicate success")
		require.NotEmpty(t, response.Message, "Success response should have a message")
	} else {
		require.False(t, response.Success, "Response should indicate failure")
		require.NotEmpty(t, response.Message, "Error response should have a message")
		require.NotEmpty(t, response.Code, "Error response should have an error code")
	}
}

// AssertUserStructure validates user data structure in response
func AssertUserStructure(t *testing.T, userData interface{}) {
	userMap, ok := userData.(map[string]interface{})
	require.True(t, ok, "User data should be a map")

	// Check required fields
	require.Contains(t, userMap, "id", "User should have ID")
	require.Contains(t, userMap, "firstName", "User should have first name")
	require.Contains(t, userMap, "lastName", "User should have last name")
	require.Contains(t, userMap, "email", "User should have email")
	require.Contains(t, userMap, "isActive", "User should have active status")
	require.Contains(t, userMap, "createdAt", "User should have creation timestamp")

	// Check that password is not included
	require.NotContains(t, userMap, "password", "User data should not contain password")
}

// AssertAddressStructure validates address data structure in response
func AssertAddressStructure(t *testing.T, addressData interface{}) {
	addressMap, ok := addressData.(map[string]interface{})
	require.True(t, ok, "Address data should be a map")

	// Check required fields
	require.Contains(t, addressMap, "id", "Address should have ID")
	require.Contains(t, addressMap, "street", "Address should have street")
	require.Contains(t, addressMap, "city", "Address should have city")
	require.Contains(t, addressMap, "state", "Address should have state")
	require.Contains(t, addressMap, "zipCode", "Address should have zip code")
	require.Contains(t, addressMap, "country", "Address should have country")
	require.Contains(t, addressMap, "isDefault", "Address should have default flag")
}

// GetEmailFromToken extracts email from the test user data
func GetEmailFromToken(t *testing.T, response APIResponse) string {
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok, "Response data should be a map")

	user, ok := data["user"].(map[string]interface{})
	require.True(t, ok, "Response should contain user data")

	email, ok := user["email"].(string)
	require.True(t, ok, "User should have email")

	return email
}

// GetTokenFromResponse extracts JWT token from auth response
func GetTokenFromResponse(t *testing.T, response APIResponse) string {
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok, "Response data should be a map")

	token, ok := data["token"].(string)
	require.True(t, ok, "Response should contain token")
	require.NotEmpty(t, token, "Token should not be empty")

	return token
}
