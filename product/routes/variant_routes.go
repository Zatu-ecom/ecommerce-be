package routes

import (
	"ecommerce-be/common/db"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/handlers"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/service"

	"github.com/gin-gonic/gin"
)

// VariantModule implements the Module interface for variant routes
type VariantModule struct {
	variantHandler *handlers.VariantHandler
}

// NewVariantModule creates a new instance of VariantModule
func NewVariantModule() *VariantModule {
	variantRepo := repositories.NewVariantRepository(db.GetDB())
	productRepo := repositories.NewProductRepository(db.GetDB())

	variantService := service.NewVariantService(variantRepo, productRepo)

	return &VariantModule{
		variantHandler: handlers.NewVariantHandler(variantService),
	}
}

// RegisterRoutes registers all variant-related routes
func (m *VariantModule) RegisterRoutes(router *gin.Engine) {
	variantRoutes := router.Group("/api/products/:productId/variants")
	{
		// Public routes
		variantRoutes.GET("/find", m.variantHandler.FindVariantByOptions)
		variantRoutes.GET("/:variantId", m.variantHandler.GetVariantByID)

		// Protected routes (require seller authentication)
		sellerAuth := middleware.SellerAuth()
		variantRoutes.POST("", sellerAuth, m.variantHandler.CreateVariant)
		variantRoutes.PUT("/:variantId", sellerAuth, m.variantHandler.UpdateVariant)
		variantRoutes.PUT("/bulk", sellerAuth, m.variantHandler.BulkUpdateVariants)
		variantRoutes.DELETE("/:variantId", sellerAuth, m.variantHandler.DeleteVariant)
		variantRoutes.PATCH("/:variantId/stock", sellerAuth, m.variantHandler.UpdateVariantStock)
	}
}
