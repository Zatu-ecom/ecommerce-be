package route

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/fulfillment/factory/singleton"
	"ecommerce-be/fulfillment/handler"

	"github.com/gin-gonic/gin"
)

// ShipmentModule implements the Module interface for draft planning and
// box reads: plan, manual create, list, detail, draft edits.
type ShipmentModule struct {
	shipmentHandler *handler.ShipmentHandler
}

// NewShipmentModule creates a new ShipmentModule.
func NewShipmentModule() *ShipmentModule {
	f := singleton.GetInstance()
	return &ShipmentModule{
		shipmentHandler: f.GetHandlerFactory().GetShipmentHandler(),
	}
}

// RegisterRoutes registers shipment routes under /api/fulfillment.
func (m *ShipmentModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	orders := router.Group("/api/fulfillment/orders")
	{
		orders.POST("/:orderId/plan", sellerAuth, m.shipmentHandler.PlanOrder)
		orders.POST("/:orderId/book", sellerAuth, m.shipmentHandler.BookAllShipments)
	}

	rates := router.Group("/api/fulfillment")
	{
		rates.POST("/rates", sellerAuth, m.shipmentHandler.GetRates)
	}

	shipments := router.Group("/api/fulfillment/shipments")
	{
		shipments.POST("", sellerAuth, m.shipmentHandler.CreateShipment)
		shipments.GET("", sellerAuth, m.shipmentHandler.ListShipments)
		shipments.GET("/:id", sellerAuth, m.shipmentHandler.GetShipment)
		shipments.PATCH("/:id", sellerAuth, m.shipmentHandler.UpdateShipment)
		shipments.POST("/:id/book", sellerAuth, m.shipmentHandler.BookShipment)
		shipments.POST("/:id/pickup", sellerAuth, m.shipmentHandler.SchedulePickup)
		shipments.POST("/:id/cancel", sellerAuth, m.shipmentHandler.CancelShipment)
		shipments.GET("/:id/label", sellerAuth, m.shipmentHandler.GetLabel)
		shipments.POST("/:id/confirm-address", sellerAuth, m.shipmentHandler.ConfirmAddress)
		shipments.POST("/:id/ndr", sellerAuth, m.shipmentHandler.ActNDR)
		shipments.POST("/:id/rto", sellerAuth, m.shipmentHandler.RequestRTO)
		shipments.POST("/:id/returns", sellerAuth, m.shipmentHandler.RequestReturn)
	}
}
