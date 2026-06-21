package routes

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/user/factory/singleton"
	"ecommerce-be/user/handler"

	"github.com/gin-gonic/gin"
)

// SellerSettingsModule handles seller settings routes
type SellerSettingsModule struct {
	sellerSettingsHandler *handler.SellerSettingsHandler
}

// NewSellerSettingsModule creates a new instance of SellerSettingsModule
func NewSellerSettingsModule() *SellerSettingsModule {
	f := singleton.GetInstance()
	return &SellerSettingsModule{
		sellerSettingsHandler: f.GetSellerSettingsHandler(),
	}
}

// RegisterRoutes registers seller-scoped settings routes.
// Admin list/get-by-seller routes are deferred until repo/service support exists.
func (m *SellerSettingsModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()
	sellerRoutes := router.Group(constants.APIBaseUser + "/seller/settings")
	sellerRoutes.Use(sellerAuth)
	{
		sellerRoutes.GET("", m.sellerSettingsHandler.GetSellerSettings)
		sellerRoutes.POST("", m.sellerSettingsHandler.CreateSellerSettings)
		sellerRoutes.PUT("", m.sellerSettingsHandler.UpdateSellerSettings)
	}
}
