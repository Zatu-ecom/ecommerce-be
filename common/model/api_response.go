package model

import (
	"github.com/gin-gonic/gin"
)

// Response is the standard API response format
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ErrorResponse includes additional error details
type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Errors  any    `json:"errors,omitempty"`
	Code    string `json:"code,omitempty"`
}

// SuccessResponse sends a successful API response
func SuccessResponse(c *gin.Context, statusCode int, message string, data any) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// ErrorWithValidation sends an error response with validation errors
func ErrorWithValidation(
	c *gin.Context,
	statusCode int,
	message string,
	errors []ValidationError,
	code string,
) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Message: message,
		Errors:  errors,
		Code:    code,
	})
}

// ErrorWithCode sends an error response with an error code
func ErrorWithCode(c *gin.Context, statusCode int, message string, code string) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Message: message,
		Code:    code,
	})
}

// ErrorResp sends a generic error response
func ErrorResp(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Message: message,
	})
}
