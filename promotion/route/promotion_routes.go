package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/promotion/factory/singleton"
	"ecommerce-be/promotion/handler"

	"github.com/gin-gonic/gin"
)

// PromotionModule implements the Module interface for promotion routes
type PromotionModule struct {
	promotionHandler *handler.PromotionHandler
}

// NewPromotionModule creates a new instance of PromotionModule
func NewPromotionModule() *PromotionModule {
	f := singleton.GetInstance()

	return &PromotionModule{
		promotionHandler: f.GetPromotionHandler(),
	}
}

// RegisterRoutes registers all promotion-related routes
func (m *PromotionModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	// Promotion routes - all protected (seller only)
	promotionRoutes := router.Group(constants.APIBasePromotion)
	{
		promotionRoutes.POST("", sellerAuth, m.promotionHandler.CreatePromotion)
		promotionRoutes.GET("", sellerAuth, m.promotionHandler.ListPromotions)
		promotionRoutes.GET("/:promotionId", sellerAuth, m.promotionHandler.GetPromotion)
		promotionRoutes.PUT("/:promotionId", sellerAuth, m.promotionHandler.UpdatePromotion)
		promotionRoutes.PATCH("/:promotionId/status", sellerAuth, m.promotionHandler.UpdateStatus)
		promotionRoutes.DELETE("/:promotionId", sellerAuth, m.promotionHandler.DeletePromotion)
	}
}
