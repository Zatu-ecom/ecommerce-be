package routes

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/user/factory/singleton"
	"ecommerce-be/user/handler"

	"github.com/gin-gonic/gin"
)

// SellerModule implements the Module interface for seller routes
type SellerModule struct {
	sellerHandler *handler.SellerHandler
}

// NewSellerModule creates a new instance of SellerModule
func NewSellerModule() *SellerModule {
	f := singleton.GetInstance()
	return &SellerModule{
		sellerHandler: f.GetSellerHandler(),
	}
}

// RegisterRoutes registers all seller related routes
func (m *SellerModule) RegisterRoutes(router *gin.Engine) {
	// Seller routes - /api/user/seller/*
	sellerRoutes := router.Group(constants.APIBaseUser + "/seller")
	{
		// Public route - no auth required
		sellerRoutes.POST("/register", m.sellerHandler.RegisterSeller)

		// Protected routes - require seller auth
		protected := sellerRoutes.Group("")
		protected.Use(middleware.SellerAuth())
		{
			protected.GET("/profile", m.sellerHandler.GetProfile)
			protected.PUT("/profile", m.sellerHandler.UpdateProfile)
		}
	}
}
