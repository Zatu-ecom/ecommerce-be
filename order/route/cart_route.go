package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/order/factory/singleton"
	"ecommerce-be/order/handler"

	"github.com/gin-gonic/gin"
)

// CartModule implements the Module interface for cart routes
type CartModule struct {
	cartHandler *handler.CartHandler
}

// NewCartModule creates a new instance of CartModule
func NewCartModule() *CartModule {
	f := singleton.GetInstance()
	return &CartModule{
		cartHandler: f.GetCartHandler(),
	}
}

// RegisterRoutes registers all cart-related routes
// All cart routes require customer authentication
func (m *CartModule) RegisterRoutes(router *gin.Engine) {
	customerAuth := middleware.CustomerAuth()

	// Cart routes - /api/cart/*
	cartRoutes := router.Group(constants.APIBaseOrder + "/cart")
	cartRoutes.Use(customerAuth)
	{
		// Cart operations
		cartRoutes.GET("", m.cartHandler.GetUserCart) // Get cart with full pricing
		cartRoutes.DELETE("/:cartId", m.cartHandler.DeleteCart)
		cartRoutes.POST("/item", m.cartHandler.AddToCart) // Add item to cart
	}
}
