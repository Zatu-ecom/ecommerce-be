package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/order/factory/singleton"
	"ecommerce-be/order/handler"

	"github.com/gin-gonic/gin"
)

// OrderModule implements the Module interface for order routes.
type OrderModule struct {
	orderHandler *handler.OrderHandler
}

// NewOrderModule creates a new instance of OrderModule.
func NewOrderModule() *OrderModule {
	f := singleton.GetInstance()
	return &OrderModule{
		orderHandler: f.GetOrderHandler(),
	}
}

// RegisterRoutes registers all order-related routes.
func (m *OrderModule) RegisterRoutes(router *gin.Engine) {
	customerAuth := middleware.CustomerAuth()

	orderRoutes := router.Group(constants.APIBaseOrder)
	{
		orderRoutes.POST("", customerAuth, m.orderHandler.CreateOrder)
		orderRoutes.GET("", customerAuth, m.orderHandler.ListOrders)
		orderRoutes.GET("/:id", customerAuth, m.orderHandler.GetOrderByID)
		orderRoutes.PATCH("/:id/status", customerAuth, m.orderHandler.UpdateOrderStatus)
		orderRoutes.POST("/:id/cancel", customerAuth, m.orderHandler.CancelOrder)
	}
}
