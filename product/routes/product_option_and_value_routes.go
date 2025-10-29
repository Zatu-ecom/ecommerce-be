package routes

import (
	"ecommerce-be/common/db"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/handlers"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/service"

	"github.com/gin-gonic/gin"
)

// ProductOptionModule implements the Module interface for product option routes
type ProductOptionModule struct {
	optionHandler *handlers.ProductOptionHandler
	valueHandler  *handlers.ProductOptionValueHandler
}

// NewProductOptionModule creates a new instance of ProductOptionModule
func NewProductOptionModule() *ProductOptionModule {
	optionRepo := repositories.NewProductOptionRepository(db.GetDB())
	productRepo := repositories.NewProductRepository(db.GetDB())

	optionService := service.NewProductOptionService(optionRepo, productRepo)
	valueService := service.NewProductOptionValueService(optionRepo, productRepo)

	return &ProductOptionModule{
		optionHandler: handlers.NewProductOptionHandler(optionService),
		valueHandler:  handlers.NewProductOptionValueHandler(valueService),
	}
}

// RegisterRoutes registers all product option-related routes
func (m *ProductOptionModule) RegisterRoutes(router *gin.Engine) {
	publicRoutesAuth := middleware.PublicAPIAuth()
	// Public routes (reading options)
	publicOptionRoutes := router.Group("/api/products/:productId/options")
	{
		publicOptionRoutes.GET("", publicRoutesAuth, m.optionHandler.GetAvailableOptions)
	}

	// Auth middleware for protected routes
	sellerAuth := middleware.SellerAuth()

	protectedOptionRoutes := router.Group("/api/products/:productId/options")
	protectedOptionRoutes.Use(sellerAuth)
	{
		protectedOptionRoutes.POST("", m.optionHandler.CreateOption)
		protectedOptionRoutes.PUT("/:optionId", m.optionHandler.UpdateOption)
		protectedOptionRoutes.DELETE("/:optionId", m.optionHandler.DeleteOption)
		protectedOptionRoutes.PUT("/bulk-update", m.optionHandler.BulkUpdateOptions)

		// Option value routes
		protectedOptionRoutes.POST("/:optionId/values", m.valueHandler.AddOptionValue)
		protectedOptionRoutes.PUT("/:optionId/values/:valueId", m.valueHandler.UpdateOptionValue)
		protectedOptionRoutes.DELETE("/:optionId/values/:valueId", m.valueHandler.DeleteOptionValue)
		protectedOptionRoutes.POST("/:optionId/values/bulk", m.valueHandler.BulkAddOptionValues)
		protectedOptionRoutes.PUT("/:optionId/values/bulk-update", m.valueHandler.BulkUpdateOptionValues)
	}
}
