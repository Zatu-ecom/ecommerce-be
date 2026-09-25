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
// All cart routes require customer authentication (JWT)
func (m *CartModule) RegisterRoutes(router *gin.Engine) {
	customerAuth := middleware.CustomerAuth()

	// Cart routes - /api/cart/*
	cartRoutes := router.Group(constants.APIBaseOrder + "/cart")
	{
		// Authenticated cart operations (require JWT)
		cartRoutes.Use(customerAuth)
		cartRoutes.GET("", m.cartHandler.GetUserCart)           // Get cart with full pricing
		cartRoutes.DELETE("/:cartId", m.cartHandler.DeleteCart) // Delete cart
		cartRoutes.POST("/item", m.cartHandler.AddToCart)       // Add item to cart

		// Merge route — merges guest (device) cart into authenticated user's cart.
		// Device ID is provided in the request body (not headers),
		// so standard CustomerAuth (JWT) is sufficient.
		cartRoutes.POST("/merge", m.cartHandler.MergeGuestCart)

		cartRoutes.POST("/coupon", m.cartHandler.ApplyCoupon)
		cartRoutes.DELETE("/coupon/:code", m.cartHandler.RemoveCoupon)
		cartRoutes.DELETE("/coupon", m.cartHandler.RemoveAllCoupons)
		cartRoutes.GET("/available-coupon", m.cartHandler.GetAvailableCoupons)
	}
}
