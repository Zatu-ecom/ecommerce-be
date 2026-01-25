package route

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/factory/singleton"
	"ecommerce-be/product/handler"

	"github.com/gin-gonic/gin"
)

// WishlistItemModule implements the Module interface for wishlist item routes
type WishlistItemModule struct {
	wishlistItemHandler *handler.WishlistItemHandler
}

// NewWishlistItemModule creates a new instance of WishlistItemModule
func NewWishlistItemModule() *WishlistItemModule {
	f := singleton.GetInstance()

	return &WishlistItemModule{
		wishlistItemHandler: f.GetWishlistItemHandler(),
	}
}

// RegisterRoutes registers all wishlist item-related routes
func (m *WishlistItemModule) RegisterRoutes(router *gin.Engine) {
	customerAuth := middleware.CustomerAuth()

	// Wishlist Item routes - /api/wishlist/:id/item
	wishlistItemRoutes := router.Group("/api/wishlist/:id/item")
	{
		// Customer routes (protected)
		wishlistItemRoutes.POST("", customerAuth, m.wishlistItemHandler.AddItem)
		wishlistItemRoutes.DELETE("/:itemId", customerAuth, m.wishlistItemHandler.RemoveItem)
		wishlistItemRoutes.POST("/:itemId/move", customerAuth, m.wishlistItemHandler.MoveItem)
	}
}
