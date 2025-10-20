package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"ecommerce-be/common"
	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"

	"github.com/gin-gonic/gin"
)

// VariantHandler handles HTTP requests related to product variants
type VariantHandler struct {
	variantService service.VariantService
}

// NewVariantHandler creates a new instance of VariantHandler
func NewVariantHandler(variantService service.VariantService) *VariantHandler {
	return &VariantHandler{
		variantService: variantService,
	}
}

/***********************************************
 *                GetVariantByID               *
 ***********************************************/
// GetVariantByID handles retrieving a specific variant by ID
// GET /api/products/:productId/variants/:variantId
func (h *VariantHandler) GetVariantByID(c *gin.Context) {
	// Parse and validate IDs
	productID, ok := h.parseProductID(c)
	if !ok {
		return
	}

	variantID, ok := h.parseVariantID(c)
	if !ok {
		return
	}

	// Call service
	variantResponse, err := h.variantService.GetVariantByID(productID, variantID)
	if err != nil {
		h.handleVariantError(c, err, utils.FAILED_TO_RETRIEVE_VARIANT_MSG)
		return
	}

	// Success response
	common.SuccessResponse(
		c,
		http.StatusOK,
		utils.VARIANT_RETRIEVED_MSG,
		map[string]interface{}{
			utils.VARIANT_FIELD_NAME: variantResponse,
		},
	)
}

/***********************************************
 *            FindVariantByOptions             *
 ***********************************************/
// FindVariantByOptions handles finding a variant by selected options
// GET /api/products/:productId/variants/find?color=red&size=m
func (h *VariantHandler) FindVariantByOptions(c *gin.Context) {
	// Parse and validate product ID
	productID, ok := h.parseProductID(c)
	if !ok {
		return
	}

	// Parse query parameters to get selected options
	queryParams := c.Request.URL.Query()
	optionValues := utils.ParseOptionsFromQuery(queryParams)

	// Validate options
	if len(optionValues) == 0 {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.OPTION_REQUIRED_MSG,
			utils.INVALID_OPTION_CODE,
		)
		return
	}

	// Call service
	variantResponse, err := h.variantService.FindVariantByOptions(productID, optionValues)
	if err != nil {
		h.handleVariantError(c, err, utils.FAILED_TO_FIND_VARIANT_MSG)
		return
	}

	// Success response
	common.SuccessResponse(
		c,
		http.StatusOK,
		utils.VARIANT_FOUND_MSG,
		map[string]interface{}{
			utils.VARIANT_FIELD_NAME: variantResponse,
		},
	)
}

/***********************************************
 *              CreateVariant                  *
 ***********************************************/
// CreateVariant handles creating a new variant for a product
// POST /api/products/:productId/variants
func (h *VariantHandler) CreateVariant(c *gin.Context) {
	// Parse and validate product ID
	productID, ok := h.parseProductID(c)
	if !ok {
		return
	}

	// Bind and validate request body
	var request model.CreateVariantRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ErrorResp(c, http.StatusBadRequest, utils.INVALID_REQUEST_MSG+": "+err.Error())
		return
	}

	// Call service
	variantResponse, err := h.variantService.CreateVariant(productID, &request)
	if err != nil {
		h.handleVariantError(c, err, utils.FAILED_TO_CREATE_VARIANT_MSG)
		return
	}

	// Send success response
	common.SuccessResponse(
		c,
		http.StatusCreated,
		utils.VARIANT_CREATED_MSG,
		map[string]interface{}{
			utils.VARIANT_FIELD_NAME: variantResponse,
		},
	)
}

/***********************************************
 *              UpdateVariant                  *
 ***********************************************/
// UpdateVariant handles updating an existing variant
// PUT /api/products/:productId/variants/:variantId
func (h *VariantHandler) UpdateVariant(c *gin.Context) {
	// Parse and validate IDs
	productID, ok := h.parseProductID(c)
	if !ok {
		return
	}

	variantID, ok := h.parseVariantID(c)
	if !ok {
		return
	}

	// Bind and validate request body
	var request model.UpdateVariantRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ErrorResp(c, http.StatusBadRequest, utils.INVALID_REQUEST_MSG+": "+err.Error())
		return
	}

	// Call service
	variantResponse, err := h.variantService.UpdateVariant(productID, variantID, &request)
	if err != nil {
		h.handleVariantError(c, err, utils.FAILED_TO_UPDATE_VARIANT_MSG)
		return
	}

	// Send success response
	common.SuccessResponse(
		c,
		http.StatusOK,
		utils.VARIANT_UPDATED_MSG,
		map[string]interface{}{
			utils.VARIANT_FIELD_NAME: variantResponse,
		},
	)
}

/***********************************************
 *              DeleteVariant                  *
 ***********************************************/
// DeleteVariant handles deleting a specific variant
// DELETE /api/products/:productId/variants/:variantId
func (h *VariantHandler) DeleteVariant(c *gin.Context) {
	// Parse and validate IDs
	productID, ok := h.parseProductID(c)
	if !ok {
		return
	}

	variantID, ok := h.parseVariantID(c)
	if !ok {
		return
	}

	// Call service
	if err := h.variantService.DeleteVariant(productID, variantID); err != nil {
		h.handleVariantError(c, err, utils.FAILED_TO_DELETE_VARIANT_MSG)
		return
	}

	// Return success response
	common.SuccessResponse(c, http.StatusOK, utils.VARIANT_DELETED_MSG, nil)
}

/***********************************************
 *           UpdateVariantStock                *
 ***********************************************/
// UpdateVariantStock handles updating the stock for a specific variant
// PATCH /api/products/:productId/variants/:variantId/stock
func (h *VariantHandler) UpdateVariantStock(c *gin.Context) {
	// Parse and validate IDs
	productID, ok := h.parseProductID(c)
	if !ok {
		return
	}

	variantID, ok := h.parseVariantID(c)
	if !ok {
		return
	}

	// Bind request
	var request model.UpdateVariantStockRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ErrorResp(c, http.StatusBadRequest, utils.INVALID_REQUEST_MSG)
		return
	}

	// Call service
	stockResponse, err := h.variantService.UpdateVariantStock(productID, variantID, &request)
	if err != nil {
		h.handleVariantError(c, err, utils.FAILED_TO_UPDATE_VARIANT_STOCK_MSG)
		return
	}

	// Return success response
	common.SuccessResponse(
		c,
		http.StatusOK,
		utils.VARIANT_STOCK_UPDATED_MSG,
		map[string]interface{}{
			utils.VARIANT_FIELD_NAME: stockResponse,
		},
	)
}

/***********************************************
 *           BulkUpdateVariants                *
 ***********************************************/
// BulkUpdateVariants handles updating multiple variants at once
// PUT /api/products/:productId/variants/bulk
func (h *VariantHandler) BulkUpdateVariants(c *gin.Context) {
	// Parse and validate product ID
	productID, ok := h.parseProductID(c)
	if !ok {
		return
	}

	// Bind request
	var request model.BulkUpdateVariantsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ErrorResp(c, http.StatusBadRequest, utils.INVALID_REQUEST_MSG)
		return
	}

	// Call service
	response, err := h.variantService.BulkUpdateVariants(productID, &request)
	if err != nil {
		h.handleVariantError(c, err, utils.FAILED_TO_BULK_UPDATE_VARIANTS_MSG)
		return
	}

	// Return success response
	common.SuccessResponse(
		c,
		http.StatusOK,
		utils.VARIANTS_BULK_UPDATED_MSG,
		map[string]interface{}{
			utils.UPDATED_COUNT_FIELD_NAME: response.UpdatedCount,
			utils.VARIANTS_FIELD_NAME:      response.Variants,
		},
	)
}

/***********************************************
 *             Helper Methods                   *
 ***********************************************/

// parseProductID extracts and validates product ID from URL params
func (h *VariantHandler) parseProductID(c *gin.Context) (uint, bool) {
	productID, err := strconv.ParseUint(c.Param(utils.PRODUCT_ID_PARAM), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.INVALID_PRODUCT_ID_MSG,
			utils.INVALID_PRODUCT_ID_CODE,
		)
		return 0, false
	}
	return uint(productID), true
}

// parseVariantID extracts and validates variant ID from URL params
func (h *VariantHandler) parseVariantID(c *gin.Context) (uint, bool) {
	variantID, err := strconv.ParseUint(c.Param(utils.VARIANT_ID_PARAM), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.INVALID_VARIANT_ID_MSG,
			utils.INVALID_VARIANT_ID_CODE,
		)
		return 0, false
	}
	return uint(variantID), true
}

// handleVariantError handles common variant-related errors
func (h *VariantHandler) handleVariantError(c *gin.Context, err error, context string) {
	errMsg := err.Error()

	switch errMsg {
	case utils.PRODUCT_NOT_FOUND_MSG:
		common.ErrorWithCode(c, http.StatusNotFound, errMsg, utils.PRODUCT_NOT_FOUND_CODE)
	case utils.VARIANT_NOT_FOUND_MSG:
		common.ErrorWithCode(c, http.StatusNotFound, errMsg, utils.VARIANT_NOT_FOUND_CODE)
	case utils.VARIANT_NOT_FOUND_WITH_OPTIONS_MSG:
		common.ErrorWithCode(
			c,
			http.StatusNotFound,
			errMsg,
			utils.VARIANT_NOT_FOUND_WITH_OPTIONS_CODE,
		)
	case utils.VARIANT_SKU_EXISTS_MSG:
		common.ErrorWithCode(c, http.StatusConflict, errMsg, utils.VARIANT_SKU_EXISTS_CODE)
	case utils.VARIANT_OPTION_COMBINATION_EXISTS_MSG:
		common.ErrorWithCode(
			c,
			http.StatusConflict,
			errMsg,
			utils.VARIANT_OPTION_COMBINATION_EXISTS_CODE,
		)
	case utils.PRODUCT_HAS_NO_OPTIONS_MSG:
		common.ErrorWithCode(c, http.StatusBadRequest, errMsg, utils.INVALID_OPTION_CODE)
	case utils.LAST_VARIANT_DELETE_NOT_ALLOWED_MSG:
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			errMsg,
			utils.LAST_VARIANT_DELETE_NOT_ALLOWED_CODE,
		)
	case utils.INVALID_STOCK_OPERATION_MSG:
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			errMsg,
			utils.INVALID_STOCK_OPERATION_CODE,
		)
	case utils.INSUFFICIENT_STOCK_FOR_OPERATION_MSG:
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			errMsg,
			utils.INSUFFICIENT_STOCK_FOR_OPERATION_CODE,
		)
	case utils.BULK_UPDATE_EMPTY_LIST_MSG:
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			errMsg,
			utils.BULK_UPDATE_EMPTY_LIST_CODE,
		)
	case utils.BULK_UPDATE_VARIANT_NOT_FOUND_MSG:
		common.ErrorWithCode(
			c,
			http.StatusNotFound,
			errMsg,
			utils.BULK_UPDATE_VARIANT_NOT_FOUND_CODE,
		)
	default:
		// Check for invalid option name errors (starts with prefix)
		if strings.HasPrefix(errMsg, utils.INVALID_OPTION_NAME_MSG) {
			common.ErrorWithCode(c, http.StatusBadRequest, errMsg, utils.INVALID_OPTION_CODE)
		} else {
			// Internal server error
			common.ErrorResp(c, http.StatusInternalServerError, context+": "+errMsg)
		}
	}
}
