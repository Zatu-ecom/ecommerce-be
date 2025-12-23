package common

import (
	"github.com/gin-gonic/gin"
)

// Response is the standard API response format
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse includes additional error details
type ErrorResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
	Code    string      `json:"code,omitempty"`
}

// ValidationError represents a single field error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ============================================================================
// Pagination
// ============================================================================

// BaseListParams contains common pagination and sorting parameters
// Embed this in query params and filter structs for list APIs
type BaseListParams struct {
	Page      int    `form:"page" json:"page"`
	PageSize  int    `form:"pageSize" json:"pageSize"`
	SortBy    string `form:"sortBy" json:"sortBy"`
	SortOrder string `form:"sortOrder" json:"sortOrder"`
}

// SetDefaults sets default values for pagination and sorting
func (b *BaseListParams) SetDefaults() {
	if b.Page <= 0 {
		b.Page = 1
	}
	if b.PageSize <= 0 {
		b.PageSize = 20
	}
	if b.PageSize > 100 {
		b.PageSize = 100
	}
	if b.SortBy == "" {
		b.SortBy = "created_at"
	}
	if b.SortOrder == "" {
		b.SortOrder = "desc"
	}
}

// PaginationResponse represents pagination information in API responses
type PaginationResponse struct {
	CurrentPage  int  `json:"currentPage"`
	TotalPages   int  `json:"totalPages"`
	TotalItems   int  `json:"totalItems"`
	ItemsPerPage int  `json:"itemsPerPage"`
	HasNext      bool `json:"hasNext"`
	HasPrev      bool `json:"hasPrev"`
}

// NewPaginationResponse creates a pagination response from params and total count
func NewPaginationResponse(page, pageSize int, total int64) PaginationResponse {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return PaginationResponse{
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalItems:   int(total),
		ItemsPerPage: pageSize,
		HasNext:      page < totalPages,
		HasPrev:      page > 1,
	}
}

// SuccessResponse sends a successful API response
func SuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
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

// ErrorResponse sends a generic error response
func ErrorResp(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Message: message,
	})
}
