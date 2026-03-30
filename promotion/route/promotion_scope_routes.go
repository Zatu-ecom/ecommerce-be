package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/promotion/factory/singleton"
	"ecommerce-be/promotion/handler"

	"github.com/gin-gonic/gin"
)

// PromotionScopeModule implements the Module interface for promotion routes
type PromotionScopeModule struct {
	promotionProductHandler    *handler.PromotionProductScopeHandler
	promotionVariantHandler    *handler.PromotionVariantScopeHandler
	promotionCategoryHandler   *handler.PromotionCategoryScopeHandler
	promotionCollectionHandler *handler.PromotionCollectionScopeHandler
}

// NewPromotionScopeModule creates a new instance of PromotionModule
func NewPromotionScopeModule() *PromotionScopeModule {
	f := singleton.GetInstance()

	return &PromotionScopeModule{
		promotionProductHandler:    f.GetPromotionProductScopeHandler(),
		promotionVariantHandler:    f.GetPromotionVariantScopeHandler(),
		promotionCategoryHandler:   f.GetPromotionCategoryScopeHandler(),
		promotionCollectionHandler: f.GetPromotionCollectionScopeHandler(),
	}
}

// RegisterRoutes registers all promotion-related routes
func (m *PromotionScopeModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	// Promotion routes - all protected (seller only)
	promotionRoutes := router.Group(constants.APIBasePromotion + "/scope")
	{
		// Product Scope Routes
		promotionRoutes.POST("/product", sellerAuth, m.promotionProductHandler.AddProducts)
		promotionRoutes.DELETE("/product", sellerAuth, m.promotionProductHandler.RemoveProducts)
		promotionRoutes.DELETE(
			"/:promotionId/product",
			sellerAuth,
			m.promotionProductHandler.RemoveAllProducts,
		)
		promotionRoutes.GET(
			"/:promotionId/product",
			sellerAuth,
			m.promotionProductHandler.GetProducts,
		)

		// Variant Scope Routes
		promotionRoutes.POST("/variant", sellerAuth, m.promotionVariantHandler.AddVariants)
		promotionRoutes.DELETE("/variant", sellerAuth, m.promotionVariantHandler.RemoveVariants)
		promotionRoutes.DELETE(
			"/:promotionId/variant",
			sellerAuth,
			m.promotionVariantHandler.RemoveAllVariants,
		)
		promotionRoutes.GET(
			"/:promotionId/variant",
			sellerAuth,
			m.promotionVariantHandler.GetVariants,
		)

		// Category Scope Routes
		promotionRoutes.POST("/category", sellerAuth, m.promotionCategoryHandler.AddCategories)
		promotionRoutes.DELETE(
			"/category",
			sellerAuth,
			m.promotionCategoryHandler.RemoveCategories,
		)
		promotionRoutes.DELETE(
			"/:promotionId/category",
			sellerAuth,
			m.promotionCategoryHandler.RemoveAllCategories,
		)
		promotionRoutes.GET(
			"/:promotionId/category",
			sellerAuth,
			m.promotionCategoryHandler.GetCategories,
		)

		// Collection Scope Routes
		promotionRoutes.POST(
			"/collection",
			sellerAuth,
			m.promotionCollectionHandler.AddCollections,
		)
		promotionRoutes.DELETE(
			"/collection",
			sellerAuth,
			m.promotionCollectionHandler.RemoveCollections,
		)
		promotionRoutes.DELETE(
			"/:promotionId/collection",
			sellerAuth,
			m.promotionCollectionHandler.RemoveAllCollections,
		)
		promotionRoutes.GET(
			"/:promotionId/collection",
			sellerAuth,
			m.promotionCollectionHandler.GetCollections,
		)
	}
}
