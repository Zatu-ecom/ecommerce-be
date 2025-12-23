package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/handler"
	"ecommerce-be/user/model"
	"ecommerce-be/user/service"
	"ecommerce-be/user/utils"

	"github.com/gin-gonic/gin"
)

// UserQueryHandler handles HTTP requests for user queries
type UserQueryHandler struct {
	*handler.BaseHandler
	userQueryService service.UserQueryService
}

// NewUserQueryHandler creates a new instance of UserQueryHandler
func NewUserQueryHandler(userQueryService service.UserQueryService) *UserQueryHandler {
	return &UserQueryHandler{
		BaseHandler:      handler.NewBaseHandler(),
		userQueryService: userQueryService,
	}
}

// ListUsers handles listing users with filters.
//
//	@Summary		List users
//	@Description	Get paginated list of users with filters
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			ids				query	string	false	"Comma-separated user IDs"
//	@Param			emails			query	string	false	"Comma-separated emails"
//	@Param			phones			query	string	false	"Comma-separated phone numbers"
//	@Param			roleIds			query	string	false	"Comma-separated role IDs"
//	@Param			sellerIds		query	string	false	"Comma-separated seller IDs (Admin only)"
//	@Param			name			query	string	false	"Search by name (partial match)"
//	@Param			isActive		query	bool	false	"Filter by active status"
//	@Param			createdFrom		query	string	false	"Created date from (RFC3339)"
//	@Param			createdTo		query	string	false	"Created date to (RFC3339)"
//	@Param			page			query	int		false	"Page number (default: 1)"
//	@Param			pageSize		query	int		false	"Page size (default: 20, max: 100)"
//	@Param			sortBy			query	string	false	"Sort by field (default: createdAt)"
//	@Param			sortOrder		query	string	false	"Sort order: asc/desc (default: desc)"
//	@Success		200				{object}	model.ListUsersResponse
//	@Router			/api/users [get]
func (h *UserQueryHandler) ListUsers(c *gin.Context) {
	// Bind raw query parameters
	var params model.ListUsersQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Convert to filter (parses comma-separated values)
	filter := params.ToFilter()

	// Set defaults for pagination and sorting
	filter.SetDefaults()

	// Get caller's sellerID from context (nil for admin)
	callerSellerID := h.getCallerSellerID(c)

	// Call service
	response, err := h.userQueryService.ListUsers(filter, callerSellerID)
	if err != nil {
		h.HandleError(c, err, utils.FailedToListUsersMsg)
		return
	}

	h.Success(c, http.StatusOK, utils.UsersRetrievedMsg, response)
}

// getCallerSellerID gets the caller's seller ID from context
func (h *UserQueryHandler) getCallerSellerID(c *gin.Context) *uint {
	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists || sellerID == 0 {
		return nil
	}
	return &sellerID
}
