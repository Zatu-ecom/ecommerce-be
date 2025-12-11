package routes

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/inventory/factory/singleton"
	"ecommerce-be/inventory/handler"

	"github.com/gin-gonic/gin"
)

// LocationModule implements the Module interface for location routes
type LocationModule struct {
	locationHandler         *handler.LocationHandler
	inventorySummaryHandler *handler.InventorySummaryHandler
}

// NewLocationModule creates a new instance of LocationModule
func NewLocationModule() *LocationModule {
	f := singleton.GetInstance()

	return &LocationModule{
		locationHandler:         f.GetLocationHandler(),
		inventorySummaryHandler: f.GetInventorySummaryHandler(),
	}
}

// RegisterRoutes registers all location-related routes
func (m *LocationModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	// Location routes - all protected (seller only)
	locationRoutes := router.Group("/api/inventory/locations")
	{
		locationRoutes.POST("", sellerAuth, m.locationHandler.CreateLocation)
		locationRoutes.GET("", sellerAuth, m.locationHandler.GetAllLocations)
		locationRoutes.GET("/summary", sellerAuth, m.inventorySummaryHandler.GetLocationsSummary)
		locationRoutes.GET("/:locationId", sellerAuth, m.locationHandler.GetLocationByID)
		locationRoutes.PUT("/:locationId", sellerAuth, m.locationHandler.UpdateLocation)
		locationRoutes.DELETE("/:locationId", sellerAuth, m.locationHandler.DeleteLocation)
	}
}
