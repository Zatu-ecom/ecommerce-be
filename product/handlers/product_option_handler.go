package handlers

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/handler"
	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"

	"github.com/gin-gonic/gin"
)

// ProductOptionHandler handles HTTP requests related to product options
type ProductOptionHandler struct {
	*handler.BaseHandler
	optionService service.ProductOptionService
}

// NewProductOptionHandler creates a new instance of ProductOptionHandler
func NewProductOptionHandler(optionService service.ProductOptionService) *ProductOptionHandler {
	return &ProductOptionHandler{
		BaseHandler:   handler.NewBaseHandler(),
		optionService: optionService,
	}
}

// CreateOption handles product option creation
func (h *ProductOptionHandler) CreateOption(c *gin.Context) {
	// Parse product ID from URL
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	// Bind request body
	var req model.ProductOptionCreateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Create option
	optionResponse, err := h.optionService.CreateOption(productID, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_CREATE_PRODUCT_OPTION_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusCreated, utils.PRODUCT_OPTION_CREATED_MSG,
		utils.PRODUCT_OPTION_FIELD_NAME, optionResponse)
}

// UpdateOption handles product option updates
func (h *ProductOptionHandler) UpdateOption(c *gin.Context) {
	// Parse product ID from URL
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	// Parse option ID from URL
	optionID, err := h.ParseUintParam(c, "optionId")
	if err != nil {
		h.HandleError(c, err, "Invalid option ID")
		return
	}

	// Bind request body
	var req model.ProductOptionUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Update option
	optionResponse, err := h.optionService.UpdateOption(productID, optionID, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_UPDATE_PRODUCT_OPTION_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusOK, utils.PRODUCT_OPTION_UPDATED_MSG,
		utils.PRODUCT_OPTION_FIELD_NAME, optionResponse)
}

// DeleteOption handles product option deletion
func (h *ProductOptionHandler) DeleteOption(c *gin.Context) {
	// Parse product ID from URL
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	// Parse option ID from URL
	optionID, err := h.ParseUintParam(c, "optionId")
	if err != nil {
		h.HandleError(c, err, "Invalid option ID")
		return
	}

	// Delete option
	err = h.optionService.DeleteOption(productID, optionID)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_DELETE_PRODUCT_OPTION_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.PRODUCT_OPTION_DELETED_MSG, nil)
}

/***********************************************
 *           GetAvailableOptions               *
 ***********************************************/
// GetAvailableOptions handles retrieving all available options for a product
// GET /api/products/:productId/options
func (h *ProductOptionHandler) GetAvailableOptions(c *gin.Context) {
	// Parse and validate product ID
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	// Extract seller ID from context (set by PublicAPIAuth middleware)
	var sellerID *uint
	if id, exists := auth.GetSellerIDFromContext(c); exists {
		sellerID = &id
	}

	// Call service
	optionsResponse, err := h.optionService.GetAvailableOptions(productID, sellerID)
	if err != nil {
		h.HandleError(c, err, "Failed to retrieve available options")
		return
	}

	// Send success response
	h.SuccessWithData(c, http.StatusOK, "Available options retrieved successfully",
		"options", optionsResponse)
}

/***********************************************
 *       BulkUpdateOptionPositions             *
 ***********************************************/
// BulkUpdateOptions handles bulk updating product options
// PUT /api/products/:productId/options/bulk-update
func (h *ProductOptionHandler) BulkUpdateOptions(c *gin.Context) {
	// Parse product ID from URL
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	// Bind request body
	var req model.ProductOptionBulkUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Call service
	response, err := h.optionService.BulkUpdateOptions(productID, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_BULK_UPDATE_OPTIONS_MSG)
		return
	}

	h.Success(c, http.StatusOK, response.Message, map[string]interface{}{
		"updatedCount": response.UpdatedCount,
	})
}

