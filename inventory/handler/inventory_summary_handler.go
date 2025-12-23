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
	inventorySummaryService        service.InventorySummaryService
	productInventorySummaryService service.ProductInventorySummaryService
}

// NewInventorySummaryHandler creates a new instance of InventorySummaryHandler
func NewInventorySummaryHandler(
	inventorySummaryService service.InventorySummaryService,
	productInventorySummaryService service.ProductInventorySummaryService,
) *InventorySummaryHandler {
	return &InventorySummaryHandler{
		BaseHandler:                    handler.NewBaseHandler(),
		inventorySummaryService:        inventorySummaryService,
		productInventorySummaryService: productInventorySummaryService,
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

// GetProductsAtLocation handles getting products with inventory summary at a specific location
func (h *InventorySummaryHandler) GetProductsAtLocation(c *gin.Context) {
	// Get seller ID from authenticated user
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Parse path and query parameters
	var params model.ProductsAtLocationParams
	if err := c.ShouldBindUri(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Parse filter parameters
	var filter model.ProductsAtLocationFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Get products at location with inventory summary
	response, err := h.productInventorySummaryService.GetProductsAtLocation(
		c, sellerID, params.LocationID, params, filter,
	)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_GET_PRODUCTS_MSG)
		return
	}

	// Return paginated response
	h.Success(c, http.StatusOK, invConstants.PRODUCTS_RETRIEVED_MSG, response)
}

// GetVariantInventoryAtLocation handles getting variant-level inventory details for a product at a specific location
func (h *InventorySummaryHandler) GetVariantInventoryAtLocation(c *gin.Context) {
	// Get seller ID from authenticated user
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Parse path parameters
	var params model.VariantInventoryParams
	if err := c.ShouldBindUri(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Parse filter query parameters
	var filter model.VariantInventoryFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.HandleValidationError(c, err)
		return
	}
	filter.SetDefaults()

	// Get variant inventory at location
	response, err := h.productInventorySummaryService.GetVariantInventoryAtLocation(
		c, sellerID, params.ProductID, params.LocationID, filter,
	)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_GET_VARIANTS_MSG)
		return
	}

	// Return response
	h.Success(c, http.StatusOK, invConstants.VARIANTS_RETRIEVED_MSG, response)
}

