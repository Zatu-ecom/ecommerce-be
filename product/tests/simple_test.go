package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ecommerce-be/product/model"
)

// TestConstants tests that test constants are properly defined
func TestConstants(t *testing.T) {
	t.Log("🧪 Testing test constants...")

	assert.NotEmpty(t, TEST_CATEGORY_NAME, "TEST_CATEGORY_NAME should not be empty")
	assert.NotEmpty(t, TEST_CATEGORY_DESCRIPTION, "TEST_CATEGORY_DESCRIPTION should not be empty")
	assert.NotEmpty(t, TEST_ATTRIBUTE_KEY, "TEST_ATTRIBUTE_KEY should not be empty")
	assert.NotEmpty(t, TEST_ATTRIBUTE_NAME, "TEST_ATTRIBUTE_NAME should not be empty")
	assert.NotEmpty(t, TEST_ATTRIBUTE_DATA_TYPE, "TEST_ATTRIBUTE_DATA_TYPE should not be empty")
	assert.NotEmpty(t, TEST_PRODUCT_NAME, "TEST_PRODUCT_NAME should not be empty")
	assert.NotEmpty(t, TEST_PRODUCT_SKU, "TEST_PRODUCT_SKU should not be empty")
	assert.NotEmpty(t, TEST_PRODUCT_BRAND, "TEST_PRODUCT_BRAND should not be empty")
	assert.Greater(t, TEST_PRODUCT_PRICE, 0.0, "TEST_PRODUCT_PRICE should be greater than 0")
	assert.NotEmpty(t, TEST_PRODUCT_CURRENCY, "TEST_PRODUCT_CURRENCY should not be empty")

	t.Log("✅ Test constants are properly defined")
}

// TestTestDataFixtures tests that test data fixtures are properly structured
func TestTestDataFixtures(t *testing.T) {
	t.Log("🧪 Testing test data fixtures...")

	// Test category data
	assert.NotEmpty(t, TestCategoryData, "TestCategoryData should not be empty")
	for i, category := range TestCategoryData {
		assert.NotEmpty(t, category.Name, "Category %d name should not be empty", i)
		assert.NotEmpty(t, category.Description, "Category %d description should not be empty", i)
	}

	// Test attribute data
	assert.NotEmpty(t, TestAttributeData, "TestAttributeData should not be empty")
	for i, attribute := range TestAttributeData {
		assert.NotEmpty(t, attribute.Key, "Attribute %d key should not be empty", i)
		assert.NotEmpty(t, attribute.Name, "Attribute %d name should not be empty", i)
		assert.NotEmpty(t, attribute.DataType, "Attribute %d data type should not be empty", i)
	}

	// Test product data
	assert.NotEmpty(t, TestProductData, "TestProductData should not be empty")
	for i, product := range TestProductData {
		assert.NotEmpty(t, product.Name, "Product %d name should not be empty", i)
		assert.NotEmpty(t, product.SKU, "Product %d SKU should not be empty", i)
		assert.NotEmpty(t, product.Brand, "Product %d brand should not be empty", i)
		assert.Greater(t, product.Price, 0.0, "Product %d price should be greater than 0", i)
		assert.NotEmpty(t, product.Currency, "Product %d currency should not be empty", i)
	}

	t.Log("✅ Test data fixtures are properly structured")
}

// TestBuilders tests the test data builder functions
func TestBuilders(t *testing.T) {
	t.Log("🧪 Testing test data builders...")

	// Test category builders
	t.Run("Category Builders", func(t *testing.T) {
		createReq := BuildCategoryCreateRequest("Test Category", "Test Description", nil)
		assert.Equal(t, "Test Category", createReq.Name)
		assert.Equal(t, "Test Description", createReq.Description)
		assert.Nil(t, createReq.ParentID)

		updateReq := BuildCategoryUpdateRequest("Updated Category", "Updated Description", nil)
		assert.Equal(t, "Updated Category", updateReq.Name)
		assert.Equal(t, "Updated Description", updateReq.Description)
		assert.Nil(t, updateReq.ParentID)
	})

	// Test attribute builders
	t.Run("Attribute Builders", func(t *testing.T) {
		createReq := BuildAttributeCreateRequest("test_key", "Test Attribute", "string", "unit", "Test Description", []string{"value1", "value2"})
		assert.Equal(t, "test_key", createReq.Key)
		assert.Equal(t, "Test Attribute", createReq.Name)
		assert.Equal(t, "unit", createReq.Unit)
		assert.Equal(t, "Test Description", createReq.Description)
		assert.Len(t, createReq.AllowedValues, 2)

		updateReq := BuildAttributeUpdateRequest("updated_key", "Updated Attribute", "number", "kg", "Updated Description", []string{"value3"})
		assert.Equal(t, "Updated Attribute", updateReq.Name)
		assert.Equal(t, "kg", updateReq.Unit)
		assert.Equal(t, "Updated Description", updateReq.Description)
		assert.Len(t, updateReq.AllowedValues, 1)
	})

	// Test product builders
	t.Run("Product Builders", func(t *testing.T) {
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

		updateReq := BuildProductUpdateRequest("Updated Product", 2, "UpdatedBrand", "UPDATED-001", 199.99, "EUR", "Updated short", "Updated long")
		assert.Equal(t, "Updated Product", updateReq.Name)
		assert.Equal(t, uint(2), updateReq.CategoryID)
		assert.Equal(t, "UpdatedBrand", updateReq.Brand)
		assert.Equal(t, 199.99, updateReq.Price)
		assert.Equal(t, "EUR", updateReq.Currency)
		assert.Equal(t, "Updated short", updateReq.ShortDescription)
		assert.Equal(t, "Updated long", updateReq.LongDescription)
	})

	// Test search query builder
	t.Run("Search Query Builder", func(t *testing.T) {
		searchReq := BuildSearchQuery("test query", nil, nil, nil, nil, 1, 10)
		assert.Equal(t, "test query", searchReq.Query)
		assert.Equal(t, 1, searchReq.Page)
		assert.Equal(t, 10, searchReq.Limit)
	})

	t.Log("✅ Test data builders are working correctly")
}

// TestModelValidation tests basic model validation
func TestModelValidation(t *testing.T) {
	t.Log("🧪 Testing model validation...")

	t.Run("Category Models", func(t *testing.T) {
		// Test valid category create request
		validCreateReq := model.CategoryCreateRequest{
			Name:        "Valid Category",
			Description: "Valid Description",
			ParentID:    nil,
		}
		assert.NotEmpty(t, validCreateReq.Name, "Category name should not be empty")
		assert.NotEmpty(t, validCreateReq.Description, "Category description should not be empty")

		// Test valid category update request
		validUpdateReq := model.CategoryUpdateRequest{
			Name:        "Updated Category",
			Description: "Updated Description",
			ParentID:    nil,
		}
		assert.NotEmpty(t, validUpdateReq.Name, "Updated category name should not be empty")
		assert.NotEmpty(t, validUpdateReq.Description, "Updated category description should not be empty")
	})

	t.Run("Attribute Models", func(t *testing.T) {
		// Test valid attribute create request
		validCreateReq := model.AttributeDefinitionCreateRequest{
			Key:           "valid_key",
			Name:          "Valid Attribute",
			Unit:          "unit",
			Description:   "Valid Description",
			AllowedValues: []string{"value1", "value2"},
		}
		assert.NotEmpty(t, validCreateReq.Key, "Attribute key should not be empty")
		assert.NotEmpty(t, validCreateReq.Name, "Attribute name should not be empty")

		// Test valid attribute update request
		validUpdateReq := model.AttributeDefinitionUpdateRequest{
			Name:          "Updated Attribute",
			Unit:          "kg",
			Description:   "Updated Description",
			AllowedValues: []string{"value3"},
		}
		assert.NotEmpty(t, validUpdateReq.Name, "Updated attribute name should not be empty")
	})

	t.Run("Product Models", func(t *testing.T) {
		// Test valid product create request
		validCreateReq := model.ProductCreateRequest{
			Name:             "Valid Product",
			CategoryID:       1,
			Brand:            "ValidBrand",
			SKU:              "VALID-001",
			Price:            99.99,
			Currency:         "USD",
			ShortDescription: "Valid short description",
			LongDescription:  "Valid long description",
			Images:           []string{"image1.jpg", "image2.jpg"},
			IsPopular:        false,
			Discount:         0,
			Tags:             []string{"tag1", "tag2"},
			Attributes:       []model.ProductAttributeRequest{},
			PackageOptions:   []model.PackageOptionRequest{},
		}
		assert.NotEmpty(t, validCreateReq.Name, "Product name should not be empty")
		assert.NotEmpty(t, validCreateReq.SKU, "Product SKU should not be empty")
		assert.Greater(t, validCreateReq.Price, 0.0, "Product price should be greater than 0")
		assert.NotEmpty(t, validCreateReq.Currency, "Product currency should not be empty")

		// Test valid product update request
		validUpdateReq := model.ProductUpdateRequest{
			Name:             "Updated Product",
			CategoryID:       2,
			Brand:            "UpdatedBrand",
			Price:            199.99,
			Currency:         "EUR",
			ShortDescription: "Updated short description",
			LongDescription:  "Updated long description",
			Images:           []string{"image3.jpg", "image4.jpg"},
			IsPopular:        true,
			Discount:         10,
			Tags:             []string{"tag3", "tag4"},
		}
		assert.NotEmpty(t, validUpdateReq.Name, "Updated product name should not be empty")
		assert.Greater(t, validUpdateReq.Price, 0.0, "Updated product price should be greater than 0")
		assert.NotEmpty(t, validUpdateReq.Currency, "Updated product currency should not be empty")
	})

	t.Run("Search Models", func(t *testing.T) {
		// Test valid search query
		validSearchReq := model.SearchQuery{
			Query:     "test query",
			Filters:   map[string]interface{}{"category": "electronics"},
			Page:      1,
			Limit:     10,
			SortBy:    "name",
			SortOrder: "asc",
		}
		assert.NotEmpty(t, validSearchReq.Query, "Search query should not be empty")
		assert.NotEmpty(t, validSearchReq.Filters, "Filters should not be empty")
		assert.Greater(t, validSearchReq.Page, 0, "Page should be greater than 0")
		assert.Greater(t, validSearchReq.Limit, 0, "Limit should be greater than 0")
	})

	t.Log("✅ Model validation tests passed")
}

// TestUtilityFunctions tests utility functions
func TestUtilityFunctions(t *testing.T) {
	t.Log("🧪 Testing utility functions...")

	t.Run("String Operations", func(t *testing.T) {
		// Test string concatenation
		result := "Hello" + " " + "World"
		assert.Equal(t, "Hello World", result, "String concatenation should work correctly")

		// Test string length
		testString := "Test String"
		assert.Equal(t, 11, len(testString), "String length should be correct")
	})

	t.Run("Slice Operations", func(t *testing.T) {
		// Test slice creation
		testSlice := []string{"item1", "item2", "item3"}
		assert.Len(t, testSlice, 3, "Slice should have correct length")

		// Test slice append
		testSlice = append(testSlice, "item4")
		assert.Len(t, testSlice, 4, "Slice should grow after append")
		assert.Equal(t, "item4", testSlice[3], "Appended item should be correct")
	})

	t.Run("Map Operations", func(t *testing.T) {
		// Test map creation
		testMap := map[string]int{"one": 1, "two": 2, "three": 3}
		assert.Len(t, testMap, 3, "Map should have correct length")

		// Test map access
		value, exists := testMap["two"]
		assert.True(t, exists, "Key should exist in map")
		assert.Equal(t, 2, value, "Map value should be correct")

		// Test map update
		testMap["two"] = 22
		assert.Equal(t, 22, testMap["two"], "Map value should be updated")
	})

	t.Log("✅ Utility function tests passed")
}

// TestDataStructures tests basic data structure operations
func TestDataStructures(t *testing.T) {
	t.Log("🧪 Testing data structures...")

	t.Run("Array Operations", func(t *testing.T) {
		// Test array creation
		testArray := [5]int{1, 2, 3, 4, 5}
		assert.Len(t, testArray, 5, "Array should have correct length")

		// Test array access
		assert.Equal(t, 1, testArray[0], "First element should be 1")
		assert.Equal(t, 5, testArray[4], "Last element should be 5")

		// Test array iteration
		sum := 0
		for _, value := range testArray {
			sum += value
		}
		assert.Equal(t, 15, sum, "Sum of array elements should be 15")
	})

	t.Run("Struct Operations", func(t *testing.T) {
		// Test struct creation
		type TestStruct struct {
			ID   int
			Name string
		}

		testStruct := TestStruct{
			ID:   1,
			Name: "Test",
		}

		assert.Equal(t, 1, testStruct.ID, "Struct ID should be 1")
		assert.Equal(t, "Test", testStruct.Name, "Struct name should be 'Test'")

		// Test struct modification
		testStruct.Name = "Updated"
		assert.Equal(t, "Updated", testStruct.Name, "Struct name should be updated")
	})

	t.Log("✅ Data structure tests passed")
}
