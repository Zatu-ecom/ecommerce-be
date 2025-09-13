package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"ecommerce-be/common"
	commonEntity "ecommerce-be/common/entity"
	"ecommerce-be/product_management/entity"
	"ecommerce-be/product_management/model"
)

// Test constants
const (
	TEST_CATEGORY_NAME        = "Electronics"
	TEST_CATEGORY_DESCRIPTION = "Electronic devices and accessories"
	TEST_ATTRIBUTE_KEY        = "color"
	TEST_ATTRIBUTE_NAME       = "Color"
	TEST_ATTRIBUTE_DATA_TYPE  = "string"
	TEST_PRODUCT_NAME         = "Test Smartphone"
	TEST_PRODUCT_SKU          = "TEST-SMART-001"
	TEST_PRODUCT_BRAND        = "TestBrand"
	TEST_PRODUCT_PRICE        = 299.99
	TEST_PRODUCT_CURRENCY     = "USD"
)

// Test data fixtures
var (
	// Test categories
	TestCategoryData = []struct {
		Name        string
		Description string
		ParentID    *uint
	}{
		{Name: "Electronics", Description: "Electronic devices and accessories", ParentID: nil},
		{Name: "Smartphones", Description: "Mobile phones and accessories", ParentID: nil},
		{Name: "Laptops", Description: "Portable computers", ParentID: nil},
		{Name: "Gaming", Description: "Gaming devices and accessories", ParentID: nil},
	}

	// Test attributes
	TestAttributeData = []struct {
		Key           string
		Name          string
		DataType      string
		Unit          string
		Description   string
		AllowedValues []string
	}{
		{
			Key:           "color",
			Name:          "Color",
			DataType:      "string",
			Unit:          "",
			Description:   "Product color",
			AllowedValues: []string{"Black", "White", "Blue", "Red"},
		},
		{
			Key:           "size",
			Name:          "Size",
			DataType:      "string",
			Unit:          "",
			Description:   "Product size",
			AllowedValues: []string{"Small", "Medium", "Large", "XL"},
		},
		{
			Key:           "weight",
			Name:          "Weight",
			DataType:      "number",
			Unit:          "kg",
			Description:   "Product weight in kilograms",
			AllowedValues: []string{},
		},
		{
			Key:           "warranty",
			Name:          "Warranty",
			DataType:      "boolean",
			Unit:          "",
			Description:   "Product warranty availability",
			AllowedValues: []string{},
		},
	}

	// Test products
	TestProductData = []struct {
		Name             string
		Brand            string
		SKU              string
		Price            float64
		Currency         string
		ShortDescription string
		LongDescription  string
		Images           []string
		InStock          bool
		IsPopular        bool
		Discount         int
		Tags             []string
	}{
		{
			Name:             "Test Smartphone",
			Brand:            "TestBrand",
			SKU:              "TEST-SMART-001",
			Price:            299.99,
			Currency:         "USD",
			ShortDescription: "High-quality smartphone for testing",
			LongDescription:  "This is a comprehensive test smartphone with all features needed for testing purposes",
			Images:           []string{"phone1.jpg", "phone2.jpg"},
			InStock:          true,
			IsPopular:        true,
			Discount:         10,
			Tags:             []string{"smartphone", "mobile", "test"},
		},
		{
			Name:             "Test Laptop",
			Brand:            "TestBrand",
			SKU:              "TEST-LAPTOP-001",
			Price:            899.99,
			Currency:         "USD",
			ShortDescription: "High-performance laptop for testing",
			LongDescription:  "This is a comprehensive test laptop with all features needed for testing purposes",
			Images:           []string{"laptop1.jpg", "laptop2.jpg"},
			InStock:          true,
			IsPopular:        false,
			Discount:         0,
			Tags:             []string{"laptop", "computer", "test"},
		},
		{
			Name:             "Test Tablet",
			Brand:            "TestBrand",
			SKU:              "TEST-TABLET-001",
			Price:            499.99,
			Currency:         "USD",
			ShortDescription: "Versatile tablet for testing",
			LongDescription:  "This is a comprehensive test tablet with all features needed for testing purposes",
			Images:           []string{"tablet1.jpg", "tablet2.jpg"},
			InStock:          true,
			IsPopular:        true,
			Discount:         5,
			Tags:             []string{"tablet", "mobile", "test"},
		},
		{
			Name:             "Test Headphones",
			Brand:            "TestBrand",
			SKU:              "TEST-HEADPHONES-001",
			Price:            199.99,
			Currency:         "USD",
			ShortDescription: "Premium headphones for testing",
			LongDescription:  "This is a comprehensive test headphones with all features needed for testing purposes",
			Images:           []string{"headphones1.jpg", "headphones2.jpg"},
			InStock:          true,
			IsPopular:        false,
			Discount:         15,
			Tags:             []string{"headphones", "audio", "test"},
		},
	}
)

// TestHelper provides common testing utilities
type TestHelper struct {
	Router *gin.Engine
	DB     *gorm.DB
}

// NewTestHelper creates a new test helper instance
func NewTestHelper(router *gin.Engine, db *gorm.DB) *TestHelper {
	return &TestHelper{
		Router: router,
		DB:     db,
	}
}

// CreateTestCategory creates a test category in the database
func (th *TestHelper) CreateTestCategory(t *testing.T, name, description string, parentID *uint) *entity.Category {
	category := &entity.Category{
		Name:        name,
		Description: description,
		ParentID:    parentID,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	err := th.DB.Create(category).Error
	require.NoError(t, err, "Failed to create test category")

	return category
}

// CreateTestAttribute creates a test attribute definition in the database
func (th *TestHelper) CreateTestAttribute(t *testing.T, key, name, dataType, unit, description string, allowedValues []string) *entity.AttributeDefinition {
	attribute := &entity.AttributeDefinition{
		Key:           key,
		Name:          name,
		Unit:          unit,
		AllowedValues: allowedValues,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	err := th.DB.Create(attribute).Error
	require.NoError(t, err, "Failed to create test attribute")

	return attribute
}

// CreateTestProduct creates a test product in the database
func (th *TestHelper) CreateTestProduct(t *testing.T, categoryID uint, name, brand, sku string, price float64, currency string) *entity.Product {
	product := &entity.Product{
		Name:             name,
		CategoryID:       categoryID,
		Brand:            brand,
		SKU:              sku,
		Price:            price,
		Currency:         currency,
		ShortDescription: "Test product description",
		LongDescription:  "This is a comprehensive test product description",
		Images:           []string{"test1.jpg", "test2.jpg"},
		InStock:          true,
		IsPopular:        false,
		Discount:         0,
		Tags:             []string{"test", "product"},
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	err := th.DB.Create(product).Error
	require.NoError(t, err, "Failed to create test product")

	return product
}

// CreateTestCategoryAttribute creates a test category attribute association
func (th *TestHelper) CreateTestCategoryAttribute(t *testing.T, categoryID, attributeID uint, isRequired, isSearchable, isFilterable bool, sortOrder int, defaultValue string) *entity.CategoryAttribute {
	catAttr := &entity.CategoryAttribute{
		CategoryID:            categoryID,
		AttributeDefinitionID: attributeID,
		IsRequired:            isRequired,
		IsSearchable:          isSearchable,
		IsFilterable:          isFilterable,
		DefaultValue:          defaultValue,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	err := th.DB.Create(catAttr).Error
	require.NoError(t, err, "Failed to create test category attribute")

	return catAttr
}

// CreateTestProductAttribute creates a test product attribute value
func (th *TestHelper) CreateTestProductAttribute(t *testing.T, productID, attributeID uint, key, value string) *entity.ProductAttribute {
	prodAttr := &entity.ProductAttribute{
		ProductID:             productID,
		AttributeDefinitionID: attributeID,
		Value:                 value,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	err := th.DB.Create(prodAttr).Error
	require.NoError(t, err, "Failed to create test product attribute")

	return prodAttr
}

// CreateTestPackageOption creates a test package option for a product
func (th *TestHelper) CreateTestPackageOption(t *testing.T, productID uint, name, description string, price float64, quantity int) *entity.PackageOption {
	pkgOpt := &entity.PackageOption{
		ProductID:   productID,
		Name:        name,
		Description: description,
		Price:       price,
		Quantity:    quantity,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	err := th.DB.Create(pkgOpt).Error
	require.NoError(t, err, "Failed to create test package option")

	return pkgOpt
}

// CleanupTestData removes all test data from the database
func (th *TestHelper) CleanupTestData(t *testing.T) {
	// Clean up in reverse order of dependencies
	_ = th.DB.Exec("TRUNCATE TABLE package_options CASCADE").Error
	_ = th.DB.Exec("TRUNCATE TABLE product_attributes CASCADE").Error
	_ = th.DB.Exec("TRUNCATE TABLE category_attributes CASCADE").Error
	_ = th.DB.Exec("TRUNCATE TABLE products CASCADE").Error
	_ = th.DB.Exec("TRUNCATE TABLE attribute_definitions CASCADE").Error
	_ = th.DB.Exec("TRUNCATE TABLE categories CASCADE").Error

	log.Println("🧹 Test data cleaned up successfully")
}

// HTTPTestHelper provides HTTP testing utilities
type HTTPTestHelper struct {
	Router *gin.Engine
}

// NewHTTPTestHelper creates a new HTTP test helper instance
func NewHTTPTestHelper(router *gin.Engine) *HTTPTestHelper {
	return &HTTPTestHelper{
		Router: router,
	}
}

// MakeRequest makes an HTTP request and returns the response
func (hth *HTTPTestHelper) MakeRequest(t *testing.T, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err, "Failed to marshal request body")
	}

	req, err := http.NewRequest(method, path, bytes.NewBuffer(reqBody))
	require.NoError(t, err, "Failed to create HTTP request")

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Create response recorder
	w := httptest.NewRecorder()

	// Serve the request
	hth.Router.ServeHTTP(w, req)

	return w
}

// AssertResponseSuccess asserts that the response indicates success
func (hth *HTTPTestHelper) AssertResponseSuccess(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	assert.Equal(t, expectedStatus, w.Code, "Expected status code %d, got %d", expectedStatus, w.Code)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal response")

	assert.True(t, response.Success, "Expected success response")
	assert.NotEmpty(t, response.Message, "Expected non-empty message")
}

// AssertResponseError asserts that the response indicates an error
func (hth *HTTPTestHelper) AssertResponseError(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedCode string) {
	assert.Equal(t, expectedStatus, w.Code, "Expected status code %d, got %d", expectedStatus, w.Code)

	var response common.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal error response")

	assert.False(t, response.Success, "Expected error response")
	assert.NotEmpty(t, response.Message, "Expected non-empty error message")
	if expectedCode != "" {
		assert.Equal(t, expectedCode, response.Code, "Expected error code %s, got %s", expectedCode, response.Code)
	}
}

// AssertValidationError asserts that the response indicates a validation error
func (hth *HTTPTestHelper) AssertValidationError(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedCode string) {
	assert.Equal(t, expectedStatus, w.Code, "Expected status code %d, got %d", expectedStatus, w.Code)

	var response common.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal validation error response")

	assert.False(t, response.Success, "Expected validation error response")
	assert.NotEmpty(t, response.Message, "Expected non-empty validation message")
	assert.NotNil(t, response.Errors, "Expected validation errors")
	if expectedCode != "" {
		assert.Equal(t, expectedCode, response.Code, "Expected validation error code %s, got %s", expectedCode, response.Code)
	}
}

// GenerateTestToken generates a test JWT token for authentication
func (hth *HTTPTestHelper) GenerateTestToken(t *testing.T, userID uint, email string) string {
	token, err := common.GenerateToken(userID, email, os.Getenv("JWT_SECRET"))
	require.NoError(t, err, "Failed to generate test token")
	return token
}

// generateTestToken is a standalone function for generating test tokens
func generateTestToken() string {
	// For testing purposes, return a simple token
	// In a real implementation, this would generate a proper JWT token
	return "test-token-12345"
}

// GetAuthHeaders returns headers with authentication token
func (hth *HTTPTestHelper) GetAuthHeaders(t *testing.T, userID uint, email string) map[string]string {
	token := hth.GenerateTestToken(t, userID, email)
	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}
}

// Test utilities for common assertions
func AssertCategoryResponse(t *testing.T, response *model.CategoryResponse, expectedName, expectedDescription string) {
	assert.NotZero(t, response.ID, "Expected non-zero category ID")
	assert.Equal(t, expectedName, response.Name, "Expected category name %s, got %s", expectedName, response.Name)
	assert.Equal(t, expectedDescription, response.Description, "Expected category description %s, got %s", expectedDescription, response.Description)
	assert.NotEmpty(t, response.CreatedAt, "Expected non-empty created at")
	assert.NotEmpty(t, response.UpdatedAt, "Expected non-empty updated at")
}

func AssertAttributeResponse(t *testing.T, response *model.AttributeDefinitionResponse, expectedKey, expectedName, expectedDataType string) {
	assert.NotZero(t, response.ID, "Expected non-zero attribute ID")
	assert.Equal(t, expectedKey, response.Key, "Expected attribute key %s, got %s", expectedKey, response.Key)
	assert.Equal(t, expectedName, response.Name, "Expected attribute name %s, got %s", expectedName, response.Name)
	assert.NotEmpty(t, response.CreatedAt, "Expected non-empty created at")
}

func AssertProductResponse(t *testing.T, response *model.ProductResponse, expectedName, expectedSKU string, expectedPrice float64) {
	assert.NotZero(t, response.ID, "Expected non-zero product ID")
	assert.Equal(t, expectedName, response.Name, "Expected product name %s, got %s", expectedName, response.Name)
	assert.Equal(t, expectedSKU, response.SKU, "Expected product SKU %s, got %s", expectedSKU, response.SKU)
	assert.Equal(t, expectedPrice, response.Price, "Expected product price %f, got %f", expectedPrice, response.Price)
	assert.NotEmpty(t, response.CreatedAt, "Expected non-empty created at")
	assert.NotEmpty(t, response.UpdatedAt, "Expected non-empty updated at")
}

// Test data builders
func BuildCategoryCreateRequest(name, description string, parentID *uint) model.CategoryCreateRequest {
	return model.CategoryCreateRequest{
		Name:        name,
		Description: description,
		ParentID:    parentID,
	}
}

func BuildCategoryUpdateRequest(name, description string, parentID *uint) model.CategoryUpdateRequest {
	return model.CategoryUpdateRequest{
		Name:        name,
		Description: description,
		ParentID:    parentID,
	}
}

func BuildAttributeCreateRequest(key, name, dataType, unit, description string, allowedValues []string) model.AttributeDefinitionCreateRequest {
	return model.AttributeDefinitionCreateRequest{
		Key:           key,
		Name:          name,
		Unit:          unit,
		Description:   description,
		AllowedValues: allowedValues,
	}
}

func BuildAttributeUpdateRequest(key, name, dataType, unit, description string, allowedValues []string) model.AttributeDefinitionUpdateRequest {
	return model.AttributeDefinitionUpdateRequest{
		Name:          name,
		Unit:          unit,
		Description:   description,
		AllowedValues: allowedValues,
	}
}

func BuildProductCreateRequest(name string, categoryID uint, brand, sku string, price float64, currency string, shortDescription, longDescription string) model.ProductCreateRequest {
	return model.ProductCreateRequest{
		Name:             name,
		CategoryID:       categoryID,
		Brand:            brand,
		SKU:              sku,
		Price:            price,
		Currency:         currency,
		ShortDescription: shortDescription,
		LongDescription:  longDescription,
		Images:           []string{"test1.jpg", "test2.jpg"},
		IsPopular:        false,
		Discount:         0,
		Tags:             []string{"test", "product"},
		Attributes: []model.ProductAttributeRequest{
			{Key: "color", Value: "Black"},
			{Key: "size", Value: "Medium"},
		},
		PackageOptions: []model.PackageOptionRequest{
			{Name: "Standard Package", Description: "Default packaging", Price: 0.0, Quantity: 1},
		},
	}
}

func BuildProductUpdateRequest(name string, categoryID uint, brand, sku string, price float64, currency string, shortDescription, longDescription string) model.ProductUpdateRequest {
	return model.ProductUpdateRequest{
		Name:             name,
		CategoryID:       categoryID,
		Brand:            brand,
		Price:            price,
		Currency:         currency,
		ShortDescription: shortDescription,
		LongDescription:  longDescription,
		Images:           []string{"test1.jpg", "test2.jpg"},
		IsPopular:        false,
		Discount:         0,
		Tags:             []string{"test", "product"},
	}
}

func BuildSearchQuery(query string, categoryID *uint, minPrice, maxPrice *float64, attributes map[string]string, page, limit int) model.SearchQuery {
	filters := make(map[string]interface{})
	if categoryID != nil {
		filters["categoryId"] = *categoryID
	}
	if minPrice != nil {
		filters["minPrice"] = *minPrice
	}
	if maxPrice != nil {
		filters["maxPrice"] = *maxPrice
	}
	if attributes != nil {
		filters["attributes"] = attributes
	}

	return model.SearchQuery{
		Query:     query,
		Filters:   filters,
		Page:      page,
		Limit:     limit,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
}
