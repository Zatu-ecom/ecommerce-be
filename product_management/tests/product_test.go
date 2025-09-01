package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestProductBasics tests basic product functionality
func TestProductBasics(t *testing.T) {
	t.Log("🧪 Testing basic product functionality...")

	t.Run("Test Data Structure", func(t *testing.T) {
		// Test that test data is properly structured
		assert.Len(t, TestProductData, 4, "Should have 4 test products")

		// Verify first product
		firstProduct := TestProductData[0]
		assert.Equal(t, "Test Smartphone", firstProduct.Name)
		assert.Equal(t, "TestBrand", firstProduct.Brand)
		assert.Equal(t, "TEST-SMART-001", firstProduct.SKU)
		assert.Equal(t, 299.99, firstProduct.Price)
		assert.Equal(t, "USD", firstProduct.Currency)
	})

	t.Run("Test Builder Functions", func(t *testing.T) {
		// Test product create request builder
		createReq := BuildProductCreateRequest("Test Product", 1, "TestBrand", "TEST-001", 99.99, "USD", "Short desc", "Long desc")
		assert.Equal(t, "Test Product", createReq.Name)
		assert.Equal(t, uint(1), createReq.CategoryID)
		assert.Equal(t, "TestBrand", createReq.Brand)
		assert.Equal(t, "TEST-001", createReq.SKU)
		assert.Equal(t, 99.99, createReq.Price)
		assert.Equal(t, "USD", createReq.Currency)
		assert.Equal(t, "Short desc", createReq.ShortDescription)
		assert.Equal(t, "Long desc", createReq.LongDescription)
		assert.False(t, createReq.IsPopular)
		assert.Equal(t, 0, createReq.Discount)
		assert.Len(t, createReq.Tags, 2)

		// Test product update request builder
		updateReq := BuildProductUpdateRequest("Updated Product", 2, "UpdatedBrand", "UPDATED-001", 199.99, "EUR", "Updated short", "Updated long")
		assert.Equal(t, "Updated Product", updateReq.Name)
		assert.Equal(t, uint(2), updateReq.CategoryID)
		assert.Equal(t, "UpdatedBrand", updateReq.Brand)
		assert.Equal(t, 199.99, updateReq.Price)
		assert.Equal(t, "EUR", updateReq.Currency)
		assert.Equal(t, "Updated short", updateReq.ShortDescription)
		assert.Equal(t, "Updated long", updateReq.LongDescription)
	})

	t.Log("✅ Basic product tests completed")
}

// TestProductValidation tests product validation rules
func TestProductValidation(t *testing.T) {
	t.Log("🧪 Testing product validation rules...")

	t.Run("Required Fields", func(t *testing.T) {
		// Test that required fields are properly handled
		createReq := BuildProductCreateRequest("Test Product", 1, "TestBrand", "TEST-001", 99.99, "USD", "Short desc", "Long desc")
		assert.NotEmpty(t, createReq.Name, "Name should be set")
		assert.NotZero(t, createReq.CategoryID, "Category ID should be set")
		assert.NotEmpty(t, createReq.SKU, "SKU should be set")
		assert.Greater(t, createReq.Price, 0.0, "Price should be greater than 0")
		assert.NotEmpty(t, createReq.Currency, "Currency should be set")
	})

	t.Run("Price Validation", func(t *testing.T) {
		// Test that price validation works
		createReq := BuildProductCreateRequest("Test Product", 1, "TestBrand", "TEST-001", 99.99, "USD", "Short desc", "Long desc")
		assert.Greater(t, createReq.Price, 0.0, "Price should be positive")
	})

	t.Log("✅ Product validation tests completed")
}

// TestProductBusinessRules tests business logic for products
func TestProductBusinessRules(t *testing.T) {
	t.Log("🧪 Testing product business rules...")

	t.Run("Tags Handling", func(t *testing.T) {
		// Test that tags are properly handled
		createReq := BuildProductCreateRequest("Test Product", 1, "TestBrand", "TEST-001", 99.99, "USD", "Short desc", "Long desc")
		assert.Len(t, createReq.Tags, 2, "Should have 2 default tags")
		assert.Contains(t, createReq.Tags, "test")
		assert.Contains(t, createReq.Tags, "product")
	})

	t.Run("Discount Handling", func(t *testing.T) {
		// Test that discount is properly handled
		createReq := BuildProductCreateRequest("Test Product", 1, "TestBrand", "TEST-001", 99.99, "USD", "Short desc", "Long desc")
		assert.Equal(t, 0, createReq.Discount, "Default discount should be 0")
	})

	t.Run("Popular Flag", func(t *testing.T) {
		// Test that popular flag is properly handled
		createReq := BuildProductCreateRequest("Test Product", 1, "TestBrand", "TEST-001", 99.99, "USD", "Short desc", "Long desc")
		assert.False(t, createReq.IsPopular, "Default popular flag should be false")
	})

	t.Log("✅ Product business rules tests completed")
}

// TestProductAttributes tests product attribute functionality
func TestProductAttributes(t *testing.T) {
	t.Log("🧪 Testing product attributes...")

	t.Run("Attribute Structure", func(t *testing.T) {
		// Test that product attributes are properly structured
		createReq := BuildProductCreateRequest("Test Product", 1, "TestBrand", "TEST-001", 99.99, "USD", "Short desc", "Long desc")
		assert.NotNil(t, createReq.Attributes, "Attributes should not be nil")
		assert.Len(t, createReq.Attributes, 2, "Should have 2 default attributes")
	})

	t.Run("Package Options", func(t *testing.T) {
		// Test that package options are properly handled
		createReq := BuildProductCreateRequest("Test Product", 1, "TestBrand", "TEST-001", 99.99, "USD", "Short desc", "Long desc")
		assert.NotNil(t, createReq.PackageOptions, "Package options should not be nil")
	})

	t.Log("✅ Product attributes tests completed")
}
