package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/order/factory/singleton"
	"ecommerce-be/order/handler"

	"github.com/gin-gonic/gin"
)

// GuestCartModule implements the Module interface for guest cart routes.
type GuestCartModule struct {
	guestCartHandler *handler.GuestCartHandler
}

// NewGuestCartModule creates a new instance of GuestCartModule.
func NewGuestCartModule() *GuestCartModule {
	f := singleton.GetInstance()
	return &GuestCartModule{
		guestCartHandler: f.GetGuestCartHandler(),
	}
}

// RegisterRoutes registers all guest cart-related routes.
// Uses PublicAPIAuth which handles:
//   - Authenticated users (via JWT) → full auth flow
//   - Guest users (no JWT) → validates X-Seller-ID header
//
// Device ID validation is done in the handler itself (reads X-Device-ID header).
func (m *GuestCartModule) RegisterRoutes(router *gin.Engine) {
	publicAPIAuth := middleware.PublicAPIAuth()

	// Guest cart routes - /api/cart/guest/*
	guestCartRoutes := router.Group(constants.APIBaseOrder + "/cart/guest")
	guestCartRoutes.Use(publicAPIAuth)
	{
		guestCartRoutes.POST(
			"/item",
			m.guestCartHandler.AddToGuestCart,
		) // Add item to guest cart
		guestCartRoutes.GET("", m.guestCartHandler.GetGuestCart)               // Get guest cart
		guestCartRoutes.DELETE("/:cartId", m.guestCartHandler.DeleteGuestCart) // Delete guest cart
	}
}
