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

// ProductOptionHandler handles HTTP requests related to product options
type ProductOptionHandler struct {
	optionService service.ProductOptionService
}

// NewProductOptionHandler creates a new instance of ProductOptionHandler
func NewProductOptionHandler(optionService service.ProductOptionService) *ProductOptionHandler {
	return &ProductOptionHandler{
		optionService: optionService,
	}
}

// CreateOption handles product option creation
func (h *ProductOptionHandler) CreateOption(c *gin.Context) {
	// Parse product ID from URL
	productID, err := strconv.ParseUint(c.Param("productId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.INVALID_PRODUCT_ID_MSG,
			utils.INVALID_PRODUCT_ID_CODE,
		)
		return
	}

	// Bind request body
	var req model.ProductOptionCreateRequest
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

	// Create option
	optionResponse, err := h.optionService.CreateOption(uint(productID), req)
	if err != nil {
		// Handle specific errors
		if err.Error() == utils.PRODUCT_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				err.Error(),
				utils.PRODUCT_NOT_FOUND_CODE,
			)
			return
		}
		if err.Error() == utils.PRODUCT_OPTION_NAME_EXISTS_MSG {
			common.ErrorWithCode(
				c,
				http.StatusConflict,
				err.Error(),
				utils.PRODUCT_OPTION_NAME_EXISTS_CODE,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_CREATE_PRODUCT_OPTION_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(
		c,
		http.StatusCreated,
		utils.PRODUCT_OPTION_CREATED_MSG,
		map[string]interface{}{
			utils.PRODUCT_OPTION_FIELD_NAME: optionResponse,
		},
	)
}

// UpdateOption handles product option updates
func (h *ProductOptionHandler) UpdateOption(c *gin.Context) {
	// Parse product ID from URL
	productID, err := strconv.ParseUint(c.Param("productId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.INVALID_PRODUCT_ID_MSG,
			utils.INVALID_PRODUCT_ID_CODE,
		)
		return
	}

	// Parse option ID from URL
	optionID, err := strconv.ParseUint(c.Param("optionId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.INVALID_OPTION_ID_MSG,
			utils.INVALID_OPTION_ID_CODE,
		)
		return
	}

	// Bind request body
	var req model.ProductOptionUpdateRequest
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

	// Update option
	optionResponse, err := h.optionService.UpdateOption(uint(productID), uint(optionID), req)
	if err != nil {
		// Handle specific errors
		if err.Error() == utils.PRODUCT_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				err.Error(),
				utils.PRODUCT_NOT_FOUND_CODE,
			)
			return
		}
		if err.Error() == utils.PRODUCT_OPTION_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				err.Error(),
				utils.PRODUCT_OPTION_NOT_FOUND_CODE,
			)
			return
		}
		if err.Error() == utils.PRODUCT_OPTION_PRODUCT_MISMATCH_MSG {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				err.Error(),
				utils.PRODUCT_OPTION_PRODUCT_MISMATCH_CODE,
			)
			return
		}
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_UPDATE_PRODUCT_OPTION_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(
		c,
		http.StatusOK,
		utils.PRODUCT_OPTION_UPDATED_MSG,
		map[string]interface{}{
			utils.PRODUCT_OPTION_FIELD_NAME: optionResponse,
		},
	)
}

// DeleteOption handles product option deletion
func (h *ProductOptionHandler) DeleteOption(c *gin.Context) {
	// Parse product ID from URL
	productID, err := strconv.ParseUint(c.Param("productId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.INVALID_PRODUCT_ID_MSG,
			utils.INVALID_PRODUCT_ID_CODE,
		)
		return
	}

	// Parse option ID from URL
	optionID, err := strconv.ParseUint(c.Param("optionId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.INVALID_OPTION_ID_MSG,
			utils.INVALID_OPTION_ID_CODE,
		)
		return
	}

	// Delete option
	err = h.optionService.DeleteOption(uint(productID), uint(optionID))
	if err != nil {
		// Handle specific errors
		if err.Error() == utils.PRODUCT_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				err.Error(),
				utils.PRODUCT_NOT_FOUND_CODE,
			)
			return
		}
		if err.Error() == utils.PRODUCT_OPTION_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				err.Error(),
				utils.PRODUCT_OPTION_NOT_FOUND_CODE,
			)
			return
		}
		if err.Error() == utils.PRODUCT_OPTION_PRODUCT_MISMATCH_MSG {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				err.Error(),
				utils.PRODUCT_OPTION_PRODUCT_MISMATCH_CODE,
			)
			return
		}

		// Check for OptionInUseError
		if _, ok := err.(*service.OptionInUseError); ok {
			var validationErrors []common.ValidationError
			validationErrors = append(validationErrors, common.ValidationError{
				Field:   "optionId",
				Message: err.Error(),
			})
			common.ErrorWithValidation(
				c,
				http.StatusBadRequest,
				err.Error(),
				validationErrors,
				utils.PRODUCT_OPTION_IN_USE_CODE,
			)
			return
		}

		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_DELETE_PRODUCT_OPTION_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(
		c,
		http.StatusOK,
		utils.PRODUCT_OPTION_DELETED_MSG,
		nil,
	)
}

/***********************************************
 *           GetAvailableOptions               *
 ***********************************************/
// GetAvailableOptions handles retrieving all available options for a product
// GET /api/products/:productId/options
func (h *ProductOptionHandler) GetAvailableOptions(c *gin.Context) {
	// Parse and validate product ID
	productID, err := strconv.ParseUint(c.Param("productId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.INVALID_PRODUCT_ID_MSG,
			utils.INVALID_PRODUCT_ID_CODE,
		)
		return
	}

	// Call service
	optionsResponse, err := h.optionService.GetAvailableOptions(uint(productID))
	if err != nil {
		// Check for product not found
		if err.Error() == utils.PRODUCT_NOT_FOUND_MSG {
			common.ErrorWithCode(
				c,
				http.StatusNotFound,
				utils.PRODUCT_NOT_FOUND_MSG,
				utils.PRODUCT_NOT_FOUND_CODE,
			)
			return
		}

		// Internal server error
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			"Failed to retrieve available options: "+err.Error(),
		)
		return
	}

	// Send success response
	common.SuccessResponse(
		c,
		http.StatusOK,
		"Available options retrieved successfully",
		map[string]interface{}{
			"options": optionsResponse,
		},
	)
}
