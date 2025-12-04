package routes

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/inventory/factory/singleton"
	"ecommerce-be/inventory/handler"

	"github.com/gin-gonic/gin"
)

// InventoryModule implements the Module interface for inventory routes
type InventoryModule struct {
	inventoryHandler *handler.InventoryHandler
}

// NewInventoryModule creates a new instance of InventoryModule
func NewInventoryModule() *InventoryModule {
	f := singleton.GetInstance()

	return &InventoryModule{
		inventoryHandler: f.GetInventoryHandler(),
	}
}

// RegisterRoutes registers all inventory-related routes
func (m *InventoryModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	// Inventory routes - all protected (seller only)
	inventoryRoutes := router.Group("/api/inventory")
	{
		// Manage inventory (quantity, reserved quantity, threshold, or physical count)
		inventoryRoutes.POST("/manage", sellerAuth, m.inventoryHandler.ManageInventory)
		
		// Bulk manage inventory (multiple items in one request)
		inventoryRoutes.POST("/manage/bulk", sellerAuth, m.inventoryHandler.BulkManageInventory)
		
		// Get inventory by variant (across all locations)
		inventoryRoutes.GET("/products/:variantId", sellerAuth, m.inventoryHandler.GetInventoryByVariant)
		
		// Get inventory by location (all variants at location)
		inventoryRoutes.GET("/locations/:locationId/inventory", sellerAuth, m.inventoryHandler.GetInventoryByLocation)

		// List inventory transactions with filters
		inventoryRoutes.GET("/transactions", sellerAuth, m.inventoryHandler.ListTransactions)
	}
}
