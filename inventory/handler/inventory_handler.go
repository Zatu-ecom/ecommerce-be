package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonErr "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/service"
	invConstants "ecommerce-be/inventory/utils/constant"

	"github.com/gin-gonic/gin"
)

// InventoryHandler handles HTTP requests related to inventory
type InventoryHandler struct {
	*handler.BaseHandler
	inventoryService      service.InventoryManageService
	inventoryQueryService service.InventoryQueryService
	transactionService    service.InventoryTransactionService
}

// NewInventoryHandler creates a new instance of InventoryHandler
func NewInventoryHandler(
	inventoryService service.InventoryManageService,
	inventoryQueryService service.InventoryQueryService,
	transactionService service.InventoryTransactionService,
) *InventoryHandler {
	return &InventoryHandler{
		BaseHandler:           handler.NewBaseHandler(),
		inventoryService:      inventoryService,
		inventoryQueryService: inventoryQueryService,
		transactionService:    transactionService,
	}
}

// ManageInventory handles inventory management requests (quantity, reserved, threshold)
func (h *InventoryHandler) ManageInventory(c *gin.Context) {
	var req model.ManageInventoryRequest

	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Get seller ID and user ID from authenticated user
	userID, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.inventoryService.ManageInventory(c, req, sellerID, userID)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_ADJUST_INVENTORY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		invConstants.INVENTORY_UPDATED_MSG,
		invConstants.INVENTORY_FIELD_NAME,
		response,
	)
}

// BulkManageInventory handles bulk inventory management requests
func (h *InventoryHandler) BulkManageInventory(c *gin.Context) {
	var req model.BulkManageInventoryRequest

	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Get seller ID and user ID from authenticated user
	userID, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.inventoryService.BulkManageInventory(c, req, sellerID, userID)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_ADJUST_INVENTORY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		invConstants.INVENTORY_UPDATED_MSG,
		invConstants.INVENTORIES_FIELD_NAME,
		response,
	)
}

// GetInventoryByVariant handles getting inventory for a specific variant
func (h *InventoryHandler) GetInventoryByVariant(c *gin.Context) {
	variantID, err := h.ParseUintParam(c, "variantId")
	if err != nil {
		h.HandleError(c, err, "Invalid variant ID")
		return
	}

	// Get seller ID from authenticated user
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	inventories, err := h.inventoryQueryService.GetInventoryByVariant(c, variantID, sellerID)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_GET_INVENTORY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		invConstants.INVENTORIES_RETRIEVED_MSG,
		invConstants.INVENTORIES_FIELD_NAME,
		inventories,
	)
}

// GetInventoryByLocation handles getting all inventory at a specific location
func (h *InventoryHandler) GetInventoryByLocation(c *gin.Context) {
	locationID, err := h.ParseUintParam(c, "locationId")
	if err != nil {
		h.HandleError(c, err, "Invalid location ID")
		return
	}

	// Get seller ID from authenticated user
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	inventories, err := h.inventoryQueryService.GetInventoryByLocation(c, locationID, sellerID)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_GET_INVENTORY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		invConstants.INVENTORIES_RETRIEVED_MSG,
		invConstants.INVENTORIES_FIELD_NAME,
		inventories,
	)
}

// ListTransactions handles listing inventory transactions with filters
func (h *InventoryHandler) ListTransactions(c *gin.Context) {
	var params model.ListTransactionsQueryParams

	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Get seller ID from authenticated user
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	// Convert query params to filter and set seller ID for isolation
	filter := params.ToFilter()
	filter.SellerID = sellerID

	response, err := h.transactionService.ListTransactions(c, filter)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_LIST_TRANSACTIONS_MSG)
		return
	}

	h.Success(c, http.StatusOK, invConstants.TRANSACTIONS_RETRIEVED_MSG, response)
}

// GetInventories handles listing inventories with filters
func (h *InventoryHandler) GetInventories(c *gin.Context) {
	var params model.GetInventoriesParam

	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Get seller ID from authenticated user
	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		h.HandleError(c, commonErr.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_CODE)
		return
	}
    
	filter := params.ToFilter()

	inventories, err := h.inventoryQueryService.GetInventories(c, &sellerID, filter)
	if err != nil {
		h.HandleError(c, err, invConstants.FAILED_TO_GET_INVENTORY_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		invConstants.INVENTORIES_RETRIEVED_MSG,
		invConstants.INVENTORIES_FIELD_NAME,
		inventories,
	)
}
