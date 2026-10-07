package handler

import (
	"net/http"

	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/handler"

	prodErrors "ecommerce-be/product/error"

	"github.com/gin-gonic/gin"
)

// AttributeHandler handles HTTP requests related to attribute definitions
type AttributeHandler struct {
	*handler.BaseHandler
	attributeService service.AttributeDefinitionService
	specService      service.PhysicalSpecService
}

// NewAttributeHandler creates a new instance of AttributeHandler
func NewAttributeHandler(
	attributeService service.AttributeDefinitionService,
	specService service.PhysicalSpecService,
) *AttributeHandler {
	return &AttributeHandler{
		BaseHandler:      handler.NewBaseHandler(),
		attributeService: attributeService,
		specService:      specService,
	}
}

// CreateAttribute handles attribute definition creation
func (h *AttributeHandler) CreateAttribute(c *gin.Context) {
	var req model.AttributeDefinitionCreateRequest

	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	attributeResponse, err := h.attributeService.CreateAttribute(c, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_CREATE_ATTRIBUTE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		utils.ATTRIBUTE_CREATED_MSG,
		utils.ATTRIBUTE_FIELD_NAME,
		attributeResponse,
	)
}

// UpdateAttribute handles attribute definition updates
func (h *AttributeHandler) UpdateAttribute(c *gin.Context) {
	attributeID, err := h.ParseUintParam(c, "attributeId")
	if err != nil {
		h.HandleError(c, err, "Invalid attribute ID")
		return
	}

	var req model.AttributeDefinitionUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	attributeResponse, err := h.attributeService.UpdateAttribute(c, attributeID, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_UPDATE_ATTRIBUTE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.ATTRIBUTE_UPDATED_MSG,
		utils.ATTRIBUTE_FIELD_NAME,
		attributeResponse,
	)
}

// DeleteAttribute handles attribute definition deletion
func (h *AttributeHandler) DeleteAttribute(c *gin.Context) {
	attributeID, err := h.ParseUintParam(c, "attributeId")
	if err != nil {
		h.HandleError(c, err, "Invalid attribute ID")
		return
	}

	err = h.attributeService.DeleteAttribute(c, attributeID)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_UPDATE_ATTRIBUTE_MSG)
		return
	}

	h.Success(c, http.StatusOK, "Attribute definition deleted successfully", nil)
}

// GetAllAttributes handles getting all attribute definitions
func (h *AttributeHandler) GetAllAttributes(c *gin.Context) {
	var sellerIDPtr *uint
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		sellerIDPtr = &sellerID
	}
	attributesResponse, err := h.attributeService.GetAllAttributes(c, sellerIDPtr)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_ATTRIBUTES_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.ATTRIBUTES_RETRIEVED_MSG, attributesResponse)
}

// GetAttributeByID handles getting an attribute definition by ID
func (h *AttributeHandler) GetAttributeByID(c *gin.Context) {
	attributeID, err := h.ParseUintParam(c, "attributeId")
	if err != nil {
		h.HandleError(c, err, "Invalid attribute ID")
		return
	}

	var sellerIDPtr *uint
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		sellerIDPtr = &sellerID
	}

	attributeResponse, err := h.attributeService.GetAttributeByID(c, attributeID, sellerIDPtr)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_ATTRIBUTES_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.ATTRIBUTES_RETRIEVED_MSG,
		utils.ATTRIBUTE_FIELD_NAME,
		attributeResponse,
	)
}

// CreateCategoryAttributeDefinition handles category-specific attribute creation
func (h *AttributeHandler) CreateCategoryAttributeDefinition(c *gin.Context) {
	var req model.AttributeDefinitionCreateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	categoryID, err := h.ParseUintParam(c, "categoryId")
	if err != nil {
		h.HandleError(c, err, "Invalid category ID")
		return
	}

	attributeResponse, err := h.attributeService.CreateCategoryAttributeDefinition(
		c,
		categoryID,
		req,
	)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_CREATE_ATTRIBUTE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		utils.ATTRIBUTE_CREATED_MSG,
		utils.ATTRIBUTE_FIELD_NAME,
		attributeResponse,
	)
}

// ListFulfillmentDefinitions serves the shippable spec catalog for the
// seller dashboard shipping-specs block.
// GET /api/product/attribute/definitions?scope=fulfillment
func (h *AttributeHandler) ListFulfillmentDefinitions(c *gin.Context) {
	if c.Query("scope") != "fulfillment" {
		h.HandleError(c, prodErrors.ErrInvalidScope, utils.INVALID_SCOPE_MSG)
		return
	}

	groups, err := h.specService.ListFulfillmentSpecs(c)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_LIST_SPEC_DEFINITIONS_MSG)
		return
	}
	if groups == nil {
		groups = []model.SpecParameterGroup{}
	}

	h.Success(
		c,
		http.StatusOK,
		utils.PHYSICAL_SPEC_DEFINITIONS_RETRIEVED_MSG,
		model.SpecCatalogResponse{Parameters: groups},
	)
}
