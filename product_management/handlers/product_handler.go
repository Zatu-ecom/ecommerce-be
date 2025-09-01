package handlers

import (
	"net/http"
	"strconv"

	"ecommerce-be/common"
	"ecommerce-be/product_management/model"
	"ecommerce-be/product_management/service"
	"ecommerce-be/product_management/utils"

	"github.com/gin-gonic/gin"
)

// ProductHandler handles HTTP requests related to products
type ProductHandler struct {
	productService service.ProductService
}

// NewProductHandler creates a new instance of ProductHandler
func NewProductHandler(productService service.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}

// CreateProduct handles product creation
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req model.ProductCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   utils.REQUEST_FIELD_NAME,
			Message: err.Error(),
		})
		common.ErrorWithValidation(c, http.StatusBadRequest, utils.VALIDATION_FAILED_MSG, validationErrors, utils.VALIDATION_ERROR_CODE)
		return
	}

	productResponse, err := h.productService.CreateProduct(req)
	if err != nil {
		if err.Error() == utils.PRODUCT_EXISTS_MSG {
			common.ErrorWithCode(c, http.StatusConflict, err.Error(), utils.PRODUCT_EXISTS_CODE)
			return
		}
		if err.Error() == utils.PRODUCT_CATEGORY_INVALID_MSG {
			common.ErrorWithCode(c, http.StatusBadRequest, err.Error(), utils.PRODUCT_CATEGORY_INVALID_CODE)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_CREATE_PRODUCT_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusCreated, utils.PRODUCT_CREATED_MSG, map[string]interface{}{
		utils.PRODUCT_FIELD_NAME: productResponse,
	})
}

// UpdateProduct handles product updates
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("productId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, "Invalid product ID", utils.VALIDATION_ERROR_CODE)
		return
	}

	var req model.ProductUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   utils.REQUEST_FIELD_NAME,
			Message: err.Error(),
		})
		common.ErrorWithValidation(c, http.StatusBadRequest, utils.VALIDATION_FAILED_MSG, validationErrors, utils.VALIDATION_ERROR_CODE)
		return
	}

	productResponse, err := h.productService.UpdateProduct(uint(productID), req)
	if err != nil {
		if err.Error() == utils.PRODUCT_NOT_FOUND_MSG {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.PRODUCT_NOT_FOUND_CODE)
			return
		}
		if err.Error() == utils.PRODUCT_EXISTS_MSG {
			common.ErrorWithCode(c, http.StatusConflict, err.Error(), utils.PRODUCT_EXISTS_CODE)
			return
		}
		if err.Error() == utils.PRODUCT_CATEGORY_INVALID_MSG {
			common.ErrorWithCode(c, http.StatusBadRequest, err.Error(), utils.PRODUCT_CATEGORY_INVALID_CODE)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_UPDATE_PRODUCT_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.PRODUCT_UPDATED_MSG, map[string]interface{}{
		utils.PRODUCT_FIELD_NAME: productResponse,
	})
}

// DeleteProduct handles product deletion
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("productId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, "Invalid product ID", utils.VALIDATION_ERROR_CODE)
		return
	}

	err = h.productService.DeleteProduct(uint(productID))
	if err != nil {
		if err.Error() == utils.PRODUCT_NOT_FOUND_MSG {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.PRODUCT_NOT_FOUND_CODE)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_DELETE_PRODUCT_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.PRODUCT_DELETED_MSG, nil)
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
	sortBy := c.DefaultQuery("sortBy", "createdAt")
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

	productsResponse, err := h.productService.GetAllProducts(page, limit, filters)
	if err != nil {
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_GET_PRODUCTS_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.PRODUCTS_RETRIEVED_MSG, productsResponse)
}

// GetProductByID handles getting a product by ID
func (h *ProductHandler) GetProductByID(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("productId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, "Invalid product ID", utils.VALIDATION_ERROR_CODE)
		return
	}

	productResponse, err := h.productService.GetProductByID(uint(productID))
	if err != nil {
		if err.Error() == utils.PRODUCT_NOT_FOUND_MSG {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.PRODUCT_NOT_FOUND_CODE)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_GET_PRODUCT_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.PRODUCT_RETRIEVED_MSG, map[string]interface{}{
		utils.PRODUCT_FIELD_NAME: productResponse,
	})
}

// SearchProducts handles product search
func (h *ProductHandler) SearchProducts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		common.ErrorWithCode(c, http.StatusBadRequest, "Search query is required", utils.VALIDATION_ERROR_CODE)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Parse additional filters
	filters := make(map[string]interface{})
	if categoryID, err := strconv.ParseUint(c.Query("categoryId"), 10, 32); err == nil && categoryID > 0 {
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

	searchResponse, err := h.productService.SearchProducts(query, filters, page, limit)
	if err != nil {
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_SEARCH_PRODUCTS_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.PRODUCTS_FOUND_MSG, searchResponse)
}

// UpdateProductStock handles product stock updates
func (h *ProductHandler) UpdateProductStock(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("productId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, "Invalid product ID", utils.VALIDATION_ERROR_CODE)
		return
	}

	var req model.ProductStockUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   utils.REQUEST_FIELD_NAME,
			Message: err.Error(),
		})
		common.ErrorWithValidation(c, http.StatusBadRequest, utils.VALIDATION_FAILED_MSG, validationErrors, utils.VALIDATION_ERROR_CODE)
		return
	}

	err = h.productService.UpdateProductStock(uint(productID), req)
	if err != nil {
		if err.Error() == utils.PRODUCT_NOT_FOUND_MSG {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.PRODUCT_NOT_FOUND_CODE)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_UPDATE_STOCK_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.STOCK_UPDATED_MSG, nil)
}

// GetProductFilters handles getting available product filters
func (h *ProductHandler) GetProductFilters(c *gin.Context) {
	var categoryID *uint
	if categoryIDStr := c.Query("categoryId"); categoryIDStr != "" {
		if parsedID, err := strconv.ParseUint(categoryIDStr, 10, 32); err == nil {
			parsed := uint(parsedID)
			categoryID = &parsed
		}
	}

	filters, err := h.productService.GetProductFilters(categoryID)
	if err != nil {
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_GET_FILTERS_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.FILTERS_RETRIEVED_MSG, map[string]interface{}{
		utils.FILTERS_FIELD_NAME: filters,
	})
}

// GetRelatedProducts handles getting related products
func (h *ProductHandler) GetRelatedProducts(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("productId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(c, http.StatusBadRequest, "Invalid product ID", utils.VALIDATION_ERROR_CODE)
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))

	relatedProductsResponse, err := h.productService.GetRelatedProducts(uint(productID), limit)
	if err != nil {
		if err.Error() == utils.PRODUCT_NOT_FOUND_MSG {
			common.ErrorWithCode(c, http.StatusNotFound, err.Error(), utils.PRODUCT_NOT_FOUND_CODE)
			return
		}
		common.ErrorResp(c, http.StatusInternalServerError, utils.FAILED_TO_GET_RELATED_PRODUCTS_MSG+": "+err.Error())
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.RELATED_PRODUCTS_RETRIEVED_MSG, relatedProductsResponse)
}
