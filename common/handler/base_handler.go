package handler

import (
	"net/http"
	"strconv"

	"ecommerce-be/common"
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"

	"github.com/gin-gonic/gin"
)

// BaseHandler provides common handler functionality
type BaseHandler struct{}

// NewBaseHandler creates a new BaseHandler instance
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}

// HandleError centralizes error handling logic
// It checks if the error is an AppError and responds accordingly
func (h *BaseHandler) HandleError(c *gin.Context, err error, defaultMessage string) {
	// Check if it's our custom AppError
	if appErr, ok := commonError.AsAppError(err); ok {
		common.ErrorWithCode(
			c,
			appErr.StatusCode,
			appErr.Message,
			appErr.Code,
		)
		return
	}

	// Default to internal server error for unknown errors
	common.ErrorResp(
		c,
		http.StatusInternalServerError,
		defaultMessage+": "+err.Error(),
	)
}

// HandleValidationError handles JSON binding validation errors
func (h *BaseHandler) HandleValidationError(c *gin.Context, err error) {
	validationErrors := []common.ValidationError{
		{
			Field:   constants.REQUEST_FIELD_NAME,
			Message: err.Error(),
		},
	}
	common.ErrorWithValidation(
		c,
		http.StatusBadRequest,
		constants.VALIDATION_FAILED_MSG,
		validationErrors,
		constants.VALIDATION_ERROR_CODE,
	)
}

// ParseUintParam parses a uint parameter from the URL
func (h *BaseHandler) ParseUintParam(c *gin.Context, paramName string) (uint, error) {
	paramValue := c.Param(paramName)
	id, err := strconv.ParseUint(paramValue, 10, 32)
	if err != nil {
		return 0, commonError.ErrInvalidID.WithMessagef("Invalid %s", paramName)
	}
	return uint(id), nil
}

// BindJSON binds JSON request body and handles validation errors
func (h *BaseHandler) BindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return err
	}
	return nil
}

// Success sends a success response
func (h *BaseHandler) Success(c *gin.Context, statusCode int, message string, data interface{}) {
	common.SuccessResponse(c, statusCode, message, data)
}

// SuccessWithData sends a success response with data wrapped in a key
func (h *BaseHandler) SuccessWithData(
	c *gin.Context,
	statusCode int,
	message string,
	dataKey string,
	data interface{},
) {
	common.SuccessResponse(c, statusCode, message, map[string]interface{}{
		dataKey: data,
	})
}
