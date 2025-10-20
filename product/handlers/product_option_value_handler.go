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

// ProductOptionValueHandler handles HTTP requests related to product option values
type ProductOptionValueHandler struct {
	valueService service.ProductOptionValueService
}

// NewProductOptionValueHandler creates a new instance of ProductOptionValueHandler
func NewProductOptionValueHandler(
	valueService service.ProductOptionValueService,
) *ProductOptionValueHandler {
	return &ProductOptionValueHandler{
		valueService: valueService,
	}
}

// parseProductAndOptionIDs parses and validates productId and optionId from URL params
func (h *ProductOptionValueHandler) parseProductAndOptionIDs(c *gin.Context) (uint, uint, error) {
	// Parse product ID
	productID, err := strconv.ParseUint(c.Param("productId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.INVALID_PRODUCT_ID_MSG,
			utils.INVALID_PRODUCT_ID_CODE,
		)
		return 0, 0, err
	}

	// Parse option ID
	optionID, err := strconv.ParseUint(c.Param("optionId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.INVALID_OPTION_ID_MSG,
			utils.INVALID_OPTION_ID_CODE,
		)
		return 0, 0, err
	}

	return uint(productID), uint(optionID), nil
}

// parseValueID parses and validates valueId from URL params
func (h *ProductOptionValueHandler) parseValueID(c *gin.Context) (uint, error) {
	valueID, err := strconv.ParseUint(c.Param("valueId"), 10, 32)
	if err != nil {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			utils.INVALID_OPTION_VALUE_ID_MSG,
			utils.INVALID_OPTION_VALUE_ID_CODE,
		)
		return 0, err
	}
	return uint(valueID), nil
}

// handleOptionValueError handles common option value errors
func (h *ProductOptionValueHandler) handleOptionValueError(c *gin.Context, err error) bool {
	if err.Error() == utils.PRODUCT_NOT_FOUND_MSG {
		common.ErrorWithCode(
			c,
			http.StatusNotFound,
			err.Error(),
			utils.PRODUCT_NOT_FOUND_CODE,
		)
		return true
	}
	if err.Error() == utils.PRODUCT_OPTION_NOT_FOUND_MSG {
		common.ErrorWithCode(
			c,
			http.StatusNotFound,
			err.Error(),
			utils.PRODUCT_OPTION_NOT_FOUND_CODE,
		)
		return true
	}
	if err.Error() == utils.PRODUCT_OPTION_VALUE_NOT_FOUND_MSG {
		common.ErrorWithCode(
			c,
			http.StatusNotFound,
			err.Error(),
			utils.PRODUCT_OPTION_VALUE_NOT_FOUND_CODE,
		)
		return true
	}
	if err.Error() == utils.PRODUCT_OPTION_PRODUCT_MISMATCH_MSG {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			err.Error(),
			utils.PRODUCT_OPTION_PRODUCT_MISMATCH_CODE,
		)
		return true
	}
	if err.Error() == utils.PRODUCT_OPTION_VALUE_OPTION_MISMATCH_MSG {
		common.ErrorWithCode(
			c,
			http.StatusBadRequest,
			err.Error(),
			utils.PRODUCT_OPTION_VALUE_OPTION_MISMATCH_CODE,
		)
		return true
	}
	if err.Error() == utils.PRODUCT_OPTION_VALUE_EXISTS_MSG {
		common.ErrorWithCode(
			c,
			http.StatusConflict,
			err.Error(),
			utils.PRODUCT_OPTION_VALUE_EXISTS_CODE,
		)
		return true
	}
	return false
}

// AddOptionValue handles adding a value to a product option
func (h *ProductOptionValueHandler) AddOptionValue(c *gin.Context) {
	// Parse IDs from URL
	productID, optionID, err := h.parseProductAndOptionIDs(c)
	if err != nil {
		return
	}

	// Bind request body
	var req model.ProductOptionValueRequest
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

	// Add option value
	valueResponse, err := h.valueService.AddOptionValue(productID, optionID, req)
	if err != nil {
		// Handle common errors
		if h.handleOptionValueError(c, err) {
			return
		}
		// Handle other errors
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_CREATE_OPTION_VALUE_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(
		c,
		http.StatusCreated,
		utils.PRODUCT_OPTION_VALUE_ADDED_MSG,
		map[string]interface{}{
			utils.OPTION_VALUE_FIELD_NAME: valueResponse,
		},
	)
}

// UpdateOptionValue handles updating a product option value
func (h *ProductOptionValueHandler) UpdateOptionValue(c *gin.Context) {
	// Parse IDs from URL
	productID, optionID, err := h.parseProductAndOptionIDs(c)
	if err != nil {
		return
	}

	valueID, err := h.parseValueID(c)
	if err != nil {
		return
	}

	// Bind request body
	var req model.ProductOptionValueUpdateRequest
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

	// Update option value
	valueResponse, err := h.valueService.UpdateOptionValue(productID, optionID, valueID, req)
	if err != nil {
		// Handle common errors
		if h.handleOptionValueError(c, err) {
			return
		}
		// Handle other errors
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_UPDATE_OPTION_VALUE_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(
		c,
		http.StatusOK,
		utils.PRODUCT_OPTION_VALUE_UPDATED_MSG,
		map[string]interface{}{
			utils.OPTION_VALUE_FIELD_NAME: valueResponse,
		},
	)
}

// DeleteOptionValue handles deleting a product option value
func (h *ProductOptionValueHandler) DeleteOptionValue(c *gin.Context) {
	// Parse IDs from URL
	productID, optionID, err := h.parseProductAndOptionIDs(c)
	if err != nil {
		return
	}

	valueID, err := h.parseValueID(c)
	if err != nil {
		return
	}

	// Delete option value
	err = h.valueService.DeleteOptionValue(productID, optionID, valueID)
	if err != nil {
		// Handle common errors
		if h.handleOptionValueError(c, err) {
			return
		}

		// Check for OptionValueInUseError
		if _, ok := err.(*service.OptionValueInUseError); ok {
			var validationErrors []common.ValidationError
			validationErrors = append(validationErrors, common.ValidationError{
				Field:   "optionValueId",
				Message: err.Error(),
			})
			common.ErrorWithValidation(
				c,
				http.StatusBadRequest,
				err.Error(),
				validationErrors,
				utils.PRODUCT_OPTION_VALUE_IN_USE_CODE,
			)
			return
		}

		// Handle other errors
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_DELETE_OPTION_VALUE_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(
		c,
		http.StatusOK,
		utils.PRODUCT_OPTION_VALUE_DELETED_MSG,
		nil,
	)
}

// BulkAddOptionValues handles bulk adding values to a product option
func (h *ProductOptionValueHandler) BulkAddOptionValues(c *gin.Context) {
	// Parse IDs from URL
	productID, optionID, err := h.parseProductAndOptionIDs(c)
	if err != nil {
		return
	}

	// Bind request body
	var req model.ProductOptionValueBulkAddRequest
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

	// Bulk add option values
	valueResponses, err := h.valueService.BulkAddOptionValues(productID, optionID, req)
	if err != nil {
		// Handle common errors
		if h.handleOptionValueError(c, err) {
			return
		}
		// Check for duplicate errors (starts with the error message)
		if len(err.Error()) >= len(utils.PRODUCT_OPTION_VALUE_EXISTS_MSG) &&
			err.Error()[:len(utils.PRODUCT_OPTION_VALUE_EXISTS_MSG)] == utils.PRODUCT_OPTION_VALUE_EXISTS_MSG {
			common.ErrorWithCode(
				c,
				http.StatusConflict,
				err.Error(),
				utils.PRODUCT_OPTION_VALUE_EXISTS_CODE,
			)
			return
		}
		if len(err.Error()) >= len(utils.PRODUCT_OPTION_VALUE_DUPLICATE_IN_BATCH_MSG) &&
			err.Error()[:len(utils.PRODUCT_OPTION_VALUE_DUPLICATE_IN_BATCH_MSG)] == utils.PRODUCT_OPTION_VALUE_DUPLICATE_IN_BATCH_MSG {
			common.ErrorWithCode(
				c,
				http.StatusBadRequest,
				err.Error(),
				utils.PRODUCT_OPTION_VALUE_EXISTS_CODE,
			)
			return
		}
		// Handle other errors
		common.ErrorResp(
			c,
			http.StatusInternalServerError,
			utils.FAILED_TO_CREATE_OPTION_VALUE_MSG+": "+err.Error(),
		)
		return
	}

	common.SuccessResponse(
		c,
		http.StatusCreated,
		utils.PRODUCT_OPTION_VALUES_ADDED_MSG,
		map[string]interface{}{
			utils.OPTION_VALUES_FIELD_NAME: valueResponses,
			utils.ADDED_COUNT_FIELD_NAME:   len(valueResponses),
		},
	)
}
