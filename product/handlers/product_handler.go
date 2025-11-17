package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/common/validator"
	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"

	productErrors "ecommerce-be/product/errors"

	"github.com/gin-gonic/gin"
)

// ProductHandler handles HTTP requests related to products
type ProductHandler struct {
	*handler.BaseHandler
	productService      service.ProductService
	productQueryService service.ProductQueryService
}

// NewProductHandler creates a new instance of ProductHandler
func NewProductHandler(
	productService service.ProductService,
	productQueryService service.ProductQueryService,
) *ProductHandler {
	return &ProductHandler{
		BaseHandler:         handler.NewBaseHandler(),
		productService:      productService,
		productQueryService: productQueryService,
	}
}

// CreateProduct handles product creation
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req model.ProductCreateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Get seller ID from context
	roleLevel, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, "Failed to validate user role")
		return
	}

	if roleLevel < constants.SELLER_ROLE_LEVEL && req.SellerID != nil {
		sellerID = *req.SellerID
	}

	if sellerID == 0 {
		h.HandleError(c, error.ErrSellerDataMissing, "Seller ID is required to create a product")
		return
	}

	productResponse, err := h.productService.CreateProduct(req, sellerID)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_CREATE_PRODUCT_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusCreated, utils.PRODUCT_CREATED_MSG,
		utils.PRODUCT_FIELD_NAME, productResponse)
}

// UpdateProduct handles product updates
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	var req model.ProductUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	if err := validator.RequireAtLeastOneNonNilPointer(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Get seller ID from context if available (for multi-tenant isolation)
	var sellerIDPtr *uint
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		sellerIDPtr = &sellerID
	}

	productResponse, err := h.productService.UpdateProduct(productID, sellerIDPtr, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_UPDATE_PRODUCT_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusOK, utils.PRODUCT_UPDATED_MSG,
		utils.PRODUCT_FIELD_NAME, productResponse)
}

// DeleteProduct handles product deletion
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	// Get seller ID from context if available (for multi-tenant isolation)
	var sellerIDPtr *uint
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		sellerIDPtr = &sellerID
	}

	err = h.productService.DeleteProduct(productID, sellerIDPtr)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_DELETE_PRODUCT_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.PRODUCT_DELETED_MSG, nil)
}

// GetAllProducts handles getting all products with filtering and pagination
func (h *ProductHandler) GetAllProducts(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	categoryID, _ := strconv.ParseUint(c.Query("categoryId"), 10, 32)
	brand := c.Query("brand")
	minPrice, _ := strconv.ParseFloat(c.Query("minPrice"), 64)
	maxPrice, _ := strconv.ParseFloat(c.Query("maxPrice"), 64)
	inStock, _ := strconv.ParseBool(c.Query("inStock"))
	isPopular, _ := strconv.ParseBool(c.Query("isPopular"))
	sortBy := c.DefaultQuery("sortBy", "created_at")
	sortOrder := c.DefaultQuery("sortOrder", "desc")

	// Build filters
	filters := make(map[string]interface{})
	if categoryID > 0 {
		filters["categoryId"] = uint(categoryID)
	}
	if brand != "" {
		filters["brand"] = brand
	}
	if minPrice > 0 {
		filters["minPrice"] = minPrice
	}
	if maxPrice > 0 {
		filters["maxPrice"] = maxPrice
	}
	if c.Query("inStock") != "" {
		filters["inStock"] = inStock
	}
	if c.Query("isPopular") != "" {
		filters["isPopular"] = isPopular
	}
	filters["sortBy"] = sortBy
	filters["sortOrder"] = sortOrder

	// Add seller ID filter if present in context (for multi-tenant isolation)
	// Seller ID will be present from PublicAPIAuth or Auth middleware
	// If not present (admin without seller context), don't filter by seller
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		filters["sellerId"] = sellerID
	}

	// Use ProductQueryService for read operations (optimized for queries)
	productsResponse, err := h.productQueryService.GetAllProducts(page, limit, filters)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.PRODUCTS_RETRIEVED_MSG, productsResponse)
}

// GetProductByID handles getting a product by ID
func (h *ProductHandler) GetProductByID(c *gin.Context) {
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	// Get seller ID from context if available (for multi-tenant isolation)
	// If seller ID exists, verify product belongs to that seller
	// If not present (admin), allow access to any product
	var sellerIDPtr *uint
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		sellerIDPtr = &sellerID
	}

	productResponse, err := h.productQueryService.GetProductByID(productID, sellerIDPtr)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_PRODUCT_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusOK, utils.PRODUCT_RETRIEVED_MSG,
		utils.PRODUCT_FIELD_NAME, productResponse)
}

// SearchProducts handles product search
func (h *ProductHandler) SearchProducts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		h.HandleError(c, error.ErrRequiredQueryParam, "Search query parameter 'q' is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Parse additional filters
	filters := make(map[string]interface{})
	if categoryID, err := strconv.ParseUint(c.Query("categoryId"), 10, 32); err == nil &&
		categoryID > 0 {
		filters["categoryId"] = uint(categoryID)
	}
	if brand := c.Query("brand"); brand != "" {
		filters["brand"] = brand
	}
	if minPrice, err := strconv.ParseFloat(c.Query("minPrice"), 64); err == nil && minPrice > 0 {
		filters["minPrice"] = minPrice
	}
	if maxPrice, err := strconv.ParseFloat(c.Query("maxPrice"), 64); err == nil && maxPrice > 0 {
		filters["maxPrice"] = maxPrice
	}

	// Add seller ID filter if present in context (for multi-tenant isolation)
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		filters["sellerId"] = sellerID
	}

	searchResponse, err := h.productService.SearchProducts(query, filters, page, limit)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_SEARCH_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.PRODUCTS_FOUND_MSG, searchResponse)
}

// GetProductFilters handles getting available product filters
func (h *ProductHandler) GetProductFilters(c *gin.Context) {
	// Get seller ID from context if available (for multi-tenant isolation)
	// If seller ID exists, get filters for that seller's products only
	// If not present (admin), get all filters
	var sellerIDPtr *uint
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		sellerIDPtr = &sellerID
	}

	filters, err := h.productService.GetProductFilters(sellerIDPtr)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_FILTERS_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusOK, utils.FILTERS_RETRIEVED_MSG,
		utils.FILTERS_FIELD_NAME, filters)
}

// GetRelatedProductsScored handles getting related products with intelligent scoring
func (h *ProductHandler) GetRelatedProductsScored(c *gin.Context) {
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	strategies := c.DefaultQuery("strategies", "all")

	// Validate limit (must be between 1 and 100)
	if limit < 1 || limit > 100 {
		h.HandleError(c, error.ErrInvalidLimit, constants.INVALID_LIMIT_MSG)
		return
	}

	// Validate strategies parameter
	validStrategies := map[string]bool{
		"all":              true,
		"same_category":    true,
		"same_brand":       true,
		"sibling_category": true,
		"parent_category":  true,
		"child_category":   true,
		"tag_matching":     true,
		"price_range":      true,
		"seller_popular":   true,
	}

	// Check if single strategy or comma-separated list
	strategyList := strings.Split(strategies, ",")
	for _, strategy := range strategyList {
		trimmedStrategy := strings.TrimSpace(strategy)
		if !validStrategies[trimmedStrategy] {
			h.HandleError(c, productErrors.ErrInvalidStrategy, utils.INVALID_STRATEGY_MSG)
			return
		}
	}

	// Extract seller ID from context (set by PublicAPIAuth middleware)
	var sellerID *uint
	if id, exists := auth.GetSellerIDFromContext(c); exists {
		sellerID = &id
	}

	// Verify product exists before getting related products
	_, err = h.productQueryService.GetProductByID(productID, sellerID)
	if err != nil {
		h.HandleError(c, productErrors.ErrProductNotFound, utils.PRODUCT_NOT_FOUND_MSG)
		return
	}

	relatedProductsResponse, err := h.productService.GetRelatedProductsScored(
		productID,
		limit,
		page,
		strategies,
		sellerID,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_RELATED_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.RELATED_PRODUCTS_RETRIEVED_MSG, relatedProductsResponse)
}
