package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/factory/singleton"
	"ecommerce-be/product/handler"

	"github.com/gin-gonic/gin"
)

// WishlistModule implements the Module interface for wishlist routes
type WishlistModule struct {
	wishlistHandler *handler.WishlistHandler
}

// NewWishlistModule creates a new instance of WishlistModule
func NewWishlistModule() *WishlistModule {
	f := singleton.GetInstance()

	return &WishlistModule{
		wishlistHandler: f.GetWishlistHandler(),
	}
}

// RegisterRoutes registers all wishlist-related routes
func (m *WishlistModule) RegisterRoutes(router *gin.Engine) {
	customerAuth := middleware.CustomerAuth()

	// Wishlist routes - /api/wishlist
	wishlistRoutes := router.Group(constants.APIBaseProduct + "/wishlist")
	{
		// Customer routes (protected)
		wishlistRoutes.GET("", customerAuth, m.wishlistHandler.GetAllWishlists)
		wishlistRoutes.POST("", customerAuth, m.wishlistHandler.CreateWishlist)
		wishlistRoutes.GET("/:id", customerAuth, m.wishlistHandler.GetWishlistByID)
		wishlistRoutes.PUT("/:id", customerAuth, m.wishlistHandler.UpdateWishlist)
		wishlistRoutes.DELETE("/:id", customerAuth, m.wishlistHandler.DeleteWishlist)
	}
}
