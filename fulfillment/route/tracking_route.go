package route

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/fulfillment/factory/singleton"
	"ecommerce-be/fulfillment/handler"

	"github.com/gin-gonic/gin"
)

// TrackingModule implements the Module interface for customer tracking,
// seller refresh, and the seller webhook audit log.
type TrackingModule struct {
	trackingHandler *handler.TrackingHandler
}

// NewTrackingModule creates a new TrackingModule.
func NewTrackingModule() *TrackingModule {
	f := singleton.GetInstance()
	return &TrackingModule{
		trackingHandler: f.GetHandlerFactory().GetTrackingHandler(),
	}
}

// RegisterRoutes registers tracking routes.
func (m *TrackingModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()
	customerAuth := middleware.CustomerAuth()

	my := router.Group("/api/fulfillment/my/orders")
	{
		my.GET("/:orderId/shipments", customerAuth, m.trackingHandler.CustomerTrack)
	}

	shipments := router.Group("/api/fulfillment/shipments")
	{
		shipments.POST("/:id/refresh", sellerAuth, m.trackingHandler.RefreshTrack)
	}

	logs := router.Group("/api/fulfillment/webhook-logs")
	{
		logs.GET("", sellerAuth, m.trackingHandler.ListWebhookLogs)
	}
}
