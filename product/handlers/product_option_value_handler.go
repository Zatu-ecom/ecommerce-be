package handlers

import (
	"net/http"

	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"

	"github.com/gin-gonic/gin"
)

// ProductOptionValueHandler handles HTTP requests related to product option values
type ProductOptionValueHandler struct {
	*BaseHandler
	valueService service.ProductOptionValueService
}

// NewProductOptionValueHandler creates a new instance of ProductOptionValueHandler
func NewProductOptionValueHandler(
	valueService service.ProductOptionValueService,
) *ProductOptionValueHandler {
	return &ProductOptionValueHandler{
		BaseHandler:  NewBaseHandler(),
		valueService: valueService,
	}
}

// AddOptionValue handles adding a value to a product option
func (h *ProductOptionValueHandler) AddOptionValue(c *gin.Context) {
	// Parse IDs from URL
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	optionID, err := h.ParseUintParam(c, "optionId")
	if err != nil {
		h.HandleError(c, err, "Invalid option ID")
		return
	}

	// Bind request body
	var req model.ProductOptionValueRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Add option value
	valueResponse, err := h.valueService.AddOptionValue(productID, optionID, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_CREATE_OPTION_VALUE_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusCreated, utils.PRODUCT_OPTION_VALUE_ADDED_MSG,
		utils.OPTION_VALUE_FIELD_NAME, valueResponse)
}

// UpdateOptionValue handles updating a product option value
func (h *ProductOptionValueHandler) UpdateOptionValue(c *gin.Context) {
	// Parse IDs from URL
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	optionID, err := h.ParseUintParam(c, "optionId")
	if err != nil {
		h.HandleError(c, err, "Invalid option ID")
		return
	}

	valueID, err := h.ParseUintParam(c, "valueId")
	if err != nil {
		h.HandleError(c, err, "Invalid value ID")
		return
	}

	// Bind request body
	var req model.ProductOptionValueUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Update option value
	valueResponse, err := h.valueService.UpdateOptionValue(productID, optionID, valueID, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_UPDATE_OPTION_VALUE_MSG)
		return
	}

	h.SuccessWithData(c, http.StatusOK, utils.PRODUCT_OPTION_VALUE_UPDATED_MSG,
		utils.OPTION_VALUE_FIELD_NAME, valueResponse)
}

// DeleteOptionValue handles deleting a product option value
func (h *ProductOptionValueHandler) DeleteOptionValue(c *gin.Context) {
	// Parse IDs from URL
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	optionID, err := h.ParseUintParam(c, "optionId")
	if err != nil {
		h.HandleError(c, err, "Invalid option ID")
		return
	}

	valueID, err := h.ParseUintParam(c, "valueId")
	if err != nil {
		h.HandleError(c, err, "Invalid value ID")
		return
	}

	// Delete option value
	err = h.valueService.DeleteOptionValue(productID, optionID, valueID)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_DELETE_OPTION_VALUE_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.PRODUCT_OPTION_VALUE_DELETED_MSG, nil)
}

// BulkAddOptionValues handles bulk adding values to a product option
func (h *ProductOptionValueHandler) BulkAddOptionValues(c *gin.Context) {
	// Parse IDs from URL
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, "Invalid product ID")
		return
	}

	optionID, err := h.ParseUintParam(c, "optionId")
	if err != nil {
		h.HandleError(c, err, "Invalid option ID")
		return
	}

	// Bind request body
	var req model.ProductOptionValueBulkAddRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Bulk add option values
	valueResponses, err := h.valueService.BulkAddOptionValues(productID, optionID, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_CREATE_OPTION_VALUE_MSG)
		return
	}

	h.Success(c, http.StatusCreated, utils.PRODUCT_OPTION_VALUES_ADDED_MSG, map[string]interface{}{
		utils.OPTION_VALUES_FIELD_NAME: valueResponses,
		utils.ADDED_COUNT_FIELD_NAME:   len(valueResponses),
	})
}
