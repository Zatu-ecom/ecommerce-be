package handler

import (
	"net/http"
	"strconv"
	"strings"

	"ecommerce-be/common"
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/log"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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
	log.ErrorWithContext(c, "Unexpected error", err)
	common.ErrorResp(
		c,
		http.StatusInternalServerError,
		defaultMessage+": "+err.Error(),
	)
}

// HandleValidationError handles JSON binding validation errors
// It extracts field-specific validation errors from Gin's validator
func (h *BaseHandler) HandleValidationError(c *gin.Context, err error) {
	var validationErrors []common.ValidationError

	// Check if it's a validator.ValidationErrors type
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validationErrs {
			// Convert field name to JSON tag name (camelCase)
			fieldName := fieldErr.Field()

			// Get custom error message based on the validation tag
			message := getValidationErrorMessage(fieldErr)

			validationErrors = append(validationErrors, common.ValidationError{
				Field:   fieldName,
				Message: message,
			})
		}
	} else {
		// Fallback for other binding errors (e.g., JSON syntax errors)
		validationErrors = []common.ValidationError{
			{
				Field:   constants.REQUEST_FIELD_NAME,
				Message: err.Error(),
			},
		}
	}

	common.ErrorWithValidation(
		c,
		http.StatusBadRequest,
		constants.VALIDATION_FAILED_MSG,
		validationErrors,
		constants.VALIDATION_ERROR_CODE,
	)
}

// getValidationErrorMessage returns a user-friendly error message based on the validation tag
func getValidationErrorMessage(fieldErr validator.FieldError) string {
	field := fieldErr.Field()
	tag := fieldErr.Tag()
	param := fieldErr.Param()

	switch tag {
	case "required":
		return field + " is required"
	case "min":
		if fieldErr.Type().String() == "string" {
			return field + " must be at least " + param + " characters long"
		}
		return field + " must be at least " + param
	case "max":
		if fieldErr.Type().String() == "string" {
			return field + " must be at most " + param + " characters long"
		}
		return field + " must be at most " + param
	case "email":
		return field + " must be a valid email address"
	case "url":
		return field + " must be a valid URL"
	case "oneof":
		return field + " must be one of: " + strings.ReplaceAll(param, " ", ", ")
	case "gt":
		return field + " must be greater than " + param
	case "gte":
		return field + " must be greater than or equal to " + param
	case "lt":
		return field + " must be less than " + param
	case "lte":
		return field + " must be less than or equal to " + param
	case "len":
		return field + " must be exactly " + param + " characters long"
	case "alphanum":
		return field + " must contain only alphanumeric characters"
	case "numeric":
		return field + " must be a number"
	case "uuid":
		return field + " must be a valid UUID"
	default:
		return field + " is invalid"
	}
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
