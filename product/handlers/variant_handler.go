package handlers

import (
	"net/http"

	"ecommerce-be/common"
	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"
	"ecommerce-be/product/utils/helper"

	"github.com/gin-gonic/gin"
)

// VariantHandler handles HTTP requests related to product variants
type VariantHandler struct {
	*handler.BaseHandler
	variantService      service.VariantService
	variantQueryService service.VariantQueryService
	variantBulkService  service.VariantBulkService
}

// NewVariantHandler creates a new instance of VariantHandler
func NewVariantHandler(
	variantService service.VariantService,
	variantQueryService service.VariantQueryService,
	variantBulkService service.VariantBulkService,
) *VariantHandler {
	return &VariantHandler{
		BaseHandler:         handler.NewBaseHandler(),
		variantService:      variantService,
		variantQueryService: variantQueryService,
		variantBulkService:  variantBulkService,
	}
}

/***********************************************
 *                GetVariantByID               *
 ***********************************************/
// GetVariantByID handles retrieving a specific variant by ID
// GET /api/products/:productId/variants/:variantId
func (h *VariantHandler) GetVariantByID(c *gin.Context) {
	// Parse and validate IDs
	productID, err := h.ParseUintParam(c, utils.PRODUCT_ID_PARAM)
	if err != nil {
		h.HandleError(c, err, "")
		return
	}

	variantID, err := h.ParseUintParam(c, utils.VARIANT_ID_PARAM)
	if err != nil {
		h.HandleError(c, err, "")
		return
	}

	// Extract seller ID from context (set by PublicAPIAuth middleware)
	sellerID, _ := auth.GetSellerIDFromContext(c)

	// Call query service
	variantResponse, err := h.variantQueryService.GetVariantByID(c, productID, variantID, sellerID)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_RETRIEVE_VARIANT_MSG)
		return
	}

	// Success response
	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.VARIANT_RETRIEVED_MSG,
		utils.VARIANT_FIELD_NAME,
		variantResponse,
	)
}

/***********************************************
 *            FindVariantByOptions             *
 ***********************************************/
// FindVariantByOptions handles finding a variant by selected options
// GET /api/products/:productId/variants/find?color=red&size=m
func (h *VariantHandler) FindVariantByOptions(c *gin.Context) {
	// Parse and validate product ID
	productID, err := h.ParseUintParam(c, utils.PRODUCT_ID_PARAM)
	if err != nil {
		h.HandleError(c, err, "")
		return
	}

	// Parse query parameters to get selected options
	queryParams := c.Request.URL.Query()
	optionValues := helper.ParseOptionsFromQuery(queryParams, []string{})

	// Validate options
	if len(optionValues) == 0 {
		h.HandleError(c, commonError.ErrValidation.WithMessage(utils.OPTION_REQUIRED_MSG), "")
		return
	}

	// Extract seller ID from context (set by PublicAPIAuth middleware)
	var sellerID *uint
	if id, exists := auth.GetSellerIDFromContext(c); exists {
		sellerID = &id
	}

	// Call query service
	variantResponse, err := h.variantQueryService.FindVariantByOptions(
		c,
		productID,
		optionValues,
		sellerID,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_FIND_VARIANT_MSG)
		return
	}

	// Success response
	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.VARIANT_FOUND_MSG,
		utils.VARIANT_FIELD_NAME,
		variantResponse,
	)
}

/***********************************************
 *              CreateVariant                  *
 ***********************************************/
// CreateVariant handles creating a new variant for a product
// POST /api/products/:productId/variants
func (h *VariantHandler) CreateVariant(c *gin.Context) {
	// Parse and validate product ID
	productID, err := h.ParseUintParam(c, utils.PRODUCT_ID_PARAM)
	if err != nil {
		h.HandleError(c, err, "")
		return
	}

	// Bind and validate request body
	var request model.CreateVariantRequest
	if err := h.BindJSON(c, &request); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerId, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Call service
	variantResponse, err := h.variantService.CreateVariant(c, productID, sellerId, &request)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_CREATE_VARIANT_MSG)
		return
	}

	// Send success response
	h.SuccessWithData(
		c,
		http.StatusCreated,
		utils.VARIANT_CREATED_MSG,
		utils.VARIANT_FIELD_NAME,
		variantResponse,
	)
}

/***********************************************
 *              UpdateVariant                  *
 ***********************************************/
// UpdateVariant handles updating an existing variant
// PUT /api/products/:productId/variants/:variantId
func (h *VariantHandler) UpdateVariant(c *gin.Context) {
	// Parse and validate IDs
	productID, err := h.ParseUintParam(c, utils.PRODUCT_ID_PARAM)
	if err != nil {
		h.HandleError(c, err, "")
		return
	}

	variantID, err := h.ParseUintParam(c, utils.VARIANT_ID_PARAM)
	if err != nil {
		h.HandleError(c, err, "")
		return
	}

	// Bind and validate request body
	var request model.UpdateVariantRequest
	if err := h.BindJSON(c, &request); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerId, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Call service
	variantResponse, err := h.variantService.UpdateVariant(
		c,
		productID,
		variantID,
		sellerId,
		&request,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_UPDATE_VARIANT_MSG)
		return
	}

	// Send success response
	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.VARIANT_UPDATED_MSG,
		utils.VARIANT_FIELD_NAME,
		variantResponse,
	)
}

/***********************************************
 *              DeleteVariant                  *
 ***********************************************/
// DeleteVariant handles deleting a specific variant
// DELETE /api/products/:productId/variants/:variantId
func (h *VariantHandler) DeleteVariant(c *gin.Context) {
	// Parse and validate IDs
	productID, err := h.ParseUintParam(c, utils.PRODUCT_ID_PARAM)
	if err != nil {
		h.HandleError(c, err, "")
		return
	}

	variantID, err := h.ParseUintParam(c, utils.VARIANT_ID_PARAM)
	if err != nil {
		h.HandleError(c, err, "")
		return
	}

	_, sellerId, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Call service
	if err := h.variantService.DeleteVariant(c, productID, variantID, sellerId); err != nil {
		h.HandleError(c, err, utils.FAILED_TO_DELETE_VARIANT_MSG)
		return
	}

	// Return success response
	h.Success(c, http.StatusOK, utils.VARIANT_DELETED_MSG, nil)
}

/***********************************************
 *           BulkUpdateVariants                *
 ***********************************************/
// BulkUpdateVariants handles updating multiple variants at once
// PUT /api/products/:productId/variants/bulk
func (h *VariantHandler) BulkUpdateVariants(c *gin.Context) {
	// Parse and validate product ID
	productID, err := h.ParseUintParam(c, utils.PRODUCT_ID_PARAM)
	if err != nil {
		h.HandleError(c, err, "")
		return
	}

	// Bind request
	var request model.BulkUpdateVariantsRequest
	if err := h.BindJSON(c, &request); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Call bulk service
	response, err := h.variantBulkService.BulkUpdateVariants(c, productID, sellerID, &request)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_BULK_UPDATE_VARIANTS_MSG)
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
 *              ListVariants                   *
 ***********************************************/
// ListVariants handles listing/filtering variants via query parameters
// Supports both formats: ?ids=1,2,3 OR ?ids=1&ids=2&ids=3
// GET /api/variants?ids=1,2,3&minPrice=100&maxPrice=500&color=red&size=M&allowPurchase=true
func (h *VariantHandler) ListVariants(c *gin.Context) {
	var request model.ListVariantsRequest

	// --- Bind remaining normal fields ---
	if err := c.ShouldBindQuery(&request); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Extract seller ID from context (set by PublicAPIAuth middleware)
	var sellerID *uint
	if id, exists := auth.GetSellerIDFromContext(c); exists {
		sellerID = &id
	}

	// Parse option filters from query params (e.g., ?color=red&size=M)
	// These are variant options (color, size, etc.) and handled separately from the struct filters
	queryParams := c.Request.URL.Query()
	defaultExcludes := []string{
		"ids", "productIds", "productId", "product_id", "sellerId", "seller_id",
		"allowPurchase", "isDefault", "is_popular", "sku", "pageSize", "page",
	}
	optionFilters := helper.ParseOptionsFromQuery(queryParams, defaultExcludes)

	// Call variant query service (same service used for other variant queries)
	response, err := h.variantQueryService.ListVariants(c, &request, sellerID, optionFilters)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_LIST_VARIANTS_MSG)
		return
	}

	// Success response with pagination metadata
	common.SuccessResponse(
		c,
		http.StatusOK,
		utils.VARIANT_RETRIEVED_MSG,
		map[string]interface{}{
			utils.VARIANTS_FIELD_NAME: response.Variants,
			"total":                   response.Total,
			"page":                    response.Page,
			"pageSize":                response.PageSize,
		},
	)
}
