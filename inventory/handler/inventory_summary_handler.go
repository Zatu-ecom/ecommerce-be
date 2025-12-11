package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/handler"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/service"
	invConstants "ecommerce-be/inventory/utils/constant"

	"github.com/gin-gonic/gin"
)

// InventorySummaryHandler handles HTTP requests related to inventory summaries
type InventorySummaryHandler struct {
	*handler.BaseHandler
	inventorySummaryService service.InventorySummaryService
}

// NewInventorySummaryHandler creates a new instance of InventorySummaryHandler
func NewInventorySummaryHandler(
	inventorySummaryService service.InventorySummaryService,
) *InventorySummaryHandler {
	return &InventorySummaryHandler{
		BaseHandler:             handler.NewBaseHandler(),
		inventorySummaryService: inventorySummaryService,
	}
}

// GetLocationsSummary handles getting all locations with inventory summary
func (h *InventorySummaryHandler) GetLocationsSummary(c *gin.Context) {
	// Get seller ID from authenticated user
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Parse query parameters using LocationsParam
	var params model.LocationsParam
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Set pagination defaults
	params.SetDefaults()

	// Convert to filter for service layer
	filter := params.ToLocationSummaryFilter()

	// Get locations with summary (includes pagination)
	summaryResponse, err := h.inventorySummaryService.GetLocationsSummary(c, sellerID, filter)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_GET_LOCATIONS_MSG)
		return
	}

	// Return paginated response
	h.Success(c, http.StatusOK, invConstants.LOCATIONS_RETRIEVED_MSG, summaryResponse)
}

