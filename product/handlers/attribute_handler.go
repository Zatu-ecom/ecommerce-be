package handlers

import (
	"net/http"
	"strconv"

	"ecommerce-be/common"
	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"

	"github.com/gin-gonic/gin"
)

// AttributeHandler handles HTTP requests related to attribute definitions
type AttributeHandler struct {
	attributeService service.AttributeDefinitionService
}

// NewAttributeHandler creates a new instance of AttributeHandler
func NewAttributeHandler(attributeService service.AttributeDefinitionService) *AttributeHandler {
	return &AttributeHandler{
		attributeService: attributeService,
	}
}

// CreateAttribute handles attribute definition creation
func (h *AttributeHandler) CreateAttribute(c *gin.Context) {
	var req model.AttributeDefinitionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   utils.REQUEST_FIELD_NAME,
			Message: err.Error(),
		})
		common.ErrorWithValidation(
			c,
			http.StatusBadRequest,
			utils.VALIDATION_FAILED_MSG,
			validationErrors,
			utils.VALIDATION_ERROR_CODE,
		)
		return
	}

	attributeResponse, err := h.attributeService.CreateAttribute(req)
	if err != nil {
		if err.Error() == utils.ATTRIBUTE_DEFINITION_EXISTS_MSG {
			common.ErrorWithCode(
				c,
				http.StatusConflict,
				err.Error(),
				utils.ATTRIBUTE_DEFINITION_EXISTS_CODE,
			)
			return
		}
		if err.Error() == utils.ATTRIBUTE_KEY_FORMAT_MSG {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				err.Error(),
				utils.ATTRIBUTE_KEY_EXISTS_CODE,
			)
			return
		}
		if err.Error() == utils.ATTRIBUTE_DATA_TYPE_INVALID_MSG {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				err.Error(),
				utils.ATTRIBUTE_DATA_TYPE_INVALID_CODE,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_CREATE_ATTRIBUTE_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(
		c,
		http.StatusCreated,
		utils.ATTRIBUTE_CREATED_MSG,
		map[string]interface{}{
			utils.ATTRIBUTE_FIELD_NAME: attributeResponse,
		},
	)
}

// UpdateAttribute handles attribute definition updates
func (h *AttributeHandler) UpdateAttribute(c *gin.Context) {
	attributeID, err := strconv.ParseUint(c.Param("attributeId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			"Invalid attribute ID",
			utils.VALIDATION_ERROR_CODE,
		)
		return
	}

	var req model.AttributeDefinitionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   utils.REQUEST_FIELD_NAME,
			Message: err.Error(),
		})
		common.ErrorWithValidation(
			c,
			http.StatusBadRequest,
			utils.VALIDATION_FAILED_MSG,
			validationErrors,
			utils.VALIDATION_ERROR_CODE,
		)
		return
	}

	attributeResponse, err := h.attributeService.UpdateAttribute(uint(attributeID), req)
	if err != nil {
		if err.Error() == utils.ATTRIBUTE_DEFINITION_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				err.Error(),
				utils.ATTRIBUTE_DEFINITION_NOT_FOUND_CODE,
			)
			return
		}
		if err.Error() == utils.ATTRIBUTE_DATA_TYPE_INVALID_MSG {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				err.Error(),
				utils.ATTRIBUTE_DATA_TYPE_INVALID_CODE,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_UPDATE_ATTRIBUTE_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.ATTRIBUTE_UPDATED_MSG, map[string]interface{}{
		utils.ATTRIBUTE_FIELD_NAME: attributeResponse,
	})
}

// DeleteAttribute handles attribute definition deletion
func (h *AttributeHandler) DeleteAttribute(c *gin.Context) {
	attributeID, err := strconv.ParseUint(c.Param("attributeId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			"Invalid attribute ID",
			utils.VALIDATION_ERROR_CODE,
		)
		return
	}

	err = h.attributeService.DeleteAttribute(uint(attributeID))
	if err != nil {
		if err.Error() == utils.ATTRIBUTE_DEFINITION_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				err.Error(),
				utils.ATTRIBUTE_DEFINITION_NOT_FOUND_CODE,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_UPDATE_ATTRIBUTE_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, "Attribute definition deleted successfully", nil)
}

// GetAllAttributes handles getting all attribute definitions
func (h *AttributeHandler) GetAllAttributes(c *gin.Context) {
	attributesResponse, err := h.attributeService.GetAllAttributes()
	if err != nil {
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_GET_ATTRIBUTES_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.ATTRIBUTES_RETRIEVED_MSG, attributesResponse)
}

// GetAttributeByID handles getting an attribute definition by ID
func (h *AttributeHandler) GetAttributeByID(c *gin.Context) {
	attributeID, err := strconv.ParseUint(c.Param("attributeId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			"Invalid attribute ID",
			utils.VALIDATION_ERROR_CODE,
		)
		return
	}

	attributeResponse, err := h.attributeService.GetAttributeByID(uint(attributeID))
	if err != nil {
		if err.Error() == utils.ATTRIBUTE_DEFINITION_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				err.Error(),
				utils.ATTRIBUTE_DEFINITION_NOT_FOUND_CODE,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_GET_ATTRIBUTES_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(c, http.StatusOK, utils.ATTRIBUTES_RETRIEVED_MSG, map[string]interface{}{
		utils.ATTRIBUTE_FIELD_NAME: attributeResponse,
	})
}

func (h *AttributeHandler) CreateCategoryAttributeDefinition(c *gin.Context) {
	var req model.AttributeDefinitionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors []common.ValidationError
		validationErrors = append(validationErrors, common.ValidationError{
			Field:   utils.REQUEST_FIELD_NAME,
			Message: err.Error(),
		})
		common.ErrorWithValidation(
			c,
			http.StatusBadRequest,
			utils.VALIDATION_FAILED_MSG,
			validationErrors,
			utils.VALIDATION_ERROR_CODE,
		)
		return
	}

	categoryID, err := strconv.ParseUint(c.Param("categoryId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			"Invalid category ID",
			utils.VALIDATION_ERROR_CODE,
		)
		return
	}
	attributeResponse, err := h.attributeService.CreateCategoryAttributeDefinition(
		uint(categoryID),
		req,
	)
	if err != nil {
		if err.Error() == utils.ATTRIBUTE_DEFINITION_EXISTS_MSG {
			common.ErrorWithCode(
				c,
				http.StatusConflict,
				err.Error(),
				utils.ATTRIBUTE_DEFINITION_EXISTS_CODE,
			)
			return
		}
		if err.Error() == utils.ATTRIBUTE_KEY_FORMAT_MSG {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				err.Error(),
				utils.ATTRIBUTE_KEY_EXISTS_CODE,
			)
			return
		}
		if err.Error() == utils.ATTRIBUTE_DATA_TYPE_INVALID_MSG {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				err.Error(),
				utils.ATTRIBUTE_DATA_TYPE_INVALID_CODE,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_CREATE_ATTRIBUTE_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(
		c,
		http.StatusCreated,
		utils.ATTRIBUTE_CREATED_MSG,
		map[string]interface{}{
			utils.ATTRIBUTE_FIELD_NAME: attributeResponse,
		},
	)
}
