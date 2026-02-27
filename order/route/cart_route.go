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
	cartRoutes := router.Group(constants.APIBaseOrder)
	cartRoutes.Use(customerAuth)
	{
		// Cart operations
		cartRoutes.GET("", m.placeholder)         // Get cart with full pricing
		cartRoutes.GET("/summary", m.placeholder) // Get cart summary (lightweight)
		cartRoutes.DELETE("", m.placeholder)      // Clear cart

		// Cart item operations
		cartRoutes.POST("/item", m.cartHandler.AddToCart) // Add item to cart
		cartRoutes.PUT("/item/:itemId", m.placeholder)    // Update item quantity
		cartRoutes.DELETE("/item/:itemId", m.placeholder) // Remove item from cart
	}
}

// placeholder is a temporary handler until the actual handler is implemented
func (m *CartModule) placeholder(c *gin.Context) {
	c.JSON(501, gin.H{
		"success": false,
		"message": "Not implemented yet",
	})
}
