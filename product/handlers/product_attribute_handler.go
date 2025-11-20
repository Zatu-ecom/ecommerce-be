package handlers

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/handler"
	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"

	"github.com/gin-gonic/gin"
)

// ProductAttributeHandler handles HTTP requests related to product attributes
type ProductAttributeHandler struct {
	*handler.BaseHandler
	productAttrService service.ProductAttributeService
}

// NewProductAttributeHandler creates a new instance of ProductAttributeHandler
func NewProductAttributeHandler(
	productAttrService service.ProductAttributeService,
) *ProductAttributeHandler {
	return &ProductAttributeHandler{
		BaseHandler:        handler.NewBaseHandler(),
		productAttrService: productAttrService,
	}
}

// AddProductAttribute handles adding an attribute to a product
// POST /api/products/:productId/attributes
func (h *ProductAttributeHandler) AddProductAttribute(c *gin.Context) {
	// Parse product ID
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_PRODUCT_ID_MSG)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Parse request body
	var req model.AddProductAttributeRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Call service
	attributeResponse, err := h.productAttrService.AddProductAttribute(
		productID,
		sellerID,
		req,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_ADD_PRODUCT_ATTRIBUTE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		utils.PRODUCT_ATTRIBUTE_ADDED_MSG,
		utils.ATTRIBUTE_FIELD_NAME,
		attributeResponse,
	)
}

// UpdateProductAttribute handles updating a product attribute
// PUT /api/products/:productId/attributes/:attributeId
func (h *ProductAttributeHandler) UpdateProductAttribute(c *gin.Context) {
	// Parse product ID
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_PRODUCT_ID_MSG)
		return
	}

	// Parse attribute ID
	attributeID, err := h.ParseUintParam(c, "attributeId")
	if err != nil {
		h.HandleError(c, err, "Invalid attribute ID")
		return
	}

	// Get seller ID from context
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Parse request body
	var req model.UpdateProductAttributeRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Call service
	attributeResponse, err := h.productAttrService.UpdateProductAttribute(
		productID,
		attributeID,
		sellerID,
		req,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_UPDATE_PRODUCT_ATTRIBUTE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.PRODUCT_ATTRIBUTE_UPDATED_MSG,
		utils.ATTRIBUTE_FIELD_NAME,
		attributeResponse,
	)
}

// DeleteProductAttribute handles deleting a product attribute
// DELETE /api/products/:productId/attributes/:attributeId
func (h *ProductAttributeHandler) DeleteProductAttribute(c *gin.Context) {
	// Parse product ID
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_PRODUCT_ID_MSG)
		return
	}

	// Parse attribute ID
	attributeID, err := h.ParseUintParam(c, "attributeId")
	if err != nil {
		h.HandleError(c, err, "Invalid attribute ID")
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Call service
	err = h.productAttrService.DeleteProductAttribute(
		productID,
		attributeID,
		sellerID,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_DELETE_PRODUCT_ATTRIBUTE_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.PRODUCT_ATTRIBUTE_DELETED_MSG, nil)
}

// GetProductAttributes handles retrieving all attributes for a product
// GET /api/products/:productId/attributes
func (h *ProductAttributeHandler) GetProductAttributes(c *gin.Context) {
	// Parse product ID
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_PRODUCT_ID_MSG)
		return
	}

	// Call service
	attributesResponse, err := h.productAttrService.GetProductAttributes(productID)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_PRODUCT_ATTRIBUTES_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.PRODUCT_ATTRIBUTES_RETRIEVED_MSG,
		"productAttributes",
		attributesResponse,
	)
}

// BulkUpdateProductAttributes handles bulk updating multiple attributes for a product
// PUT /api/products/:productId/attributes/bulk
func (h *ProductAttributeHandler) BulkUpdateProductAttributes(c *gin.Context) {
	// Parse product ID
	productID, err := h.ParseUintParam(c, "productId")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_PRODUCT_ID_MSG)
		return
	}

	// Get seller ID from context
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Parse request body
	var req model.BulkUpdateProductAttributesRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Call service
	updateResponse, err := h.productAttrService.BulkUpdateProductAttributes(
		productID,
		sellerID,
		req,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_UPDATE_PRODUCT_ATTRIBUTES_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.PRODUCT_ATTRIBUTES_BULK_UPDATED_MSG,
		"result",
		updateResponse,
	)
}
