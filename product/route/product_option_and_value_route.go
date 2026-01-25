package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/factory/singleton"
	"ecommerce-be/product/handler"

	"github.com/gin-gonic/gin"
)

// ProductOptionModule implements the Module interface for product option routes
type ProductOptionModule struct {
	optionHandler *handler.ProductOptionHandler
	valueHandler  *handler.ProductOptionValueHandler
}

// NewProductOptionModule creates a new instance of ProductOptionModule
func NewProductOptionModule() *ProductOptionModule {
	f := singleton.GetInstance()

	return &ProductOptionModule{
		optionHandler: f.GetProductOptionHandler(),
		valueHandler:  f.GetProductOptionValueHandler(),
	}
}

// RegisterRoutes registers all product option-related routes
func (m *ProductOptionModule) RegisterRoutes(router *gin.Engine) {
	publicRoutesAuth := middleware.PublicAPIAuth()
	// Public routes (reading options) - /api/product/:productId/option
	publicOptionRoutes := router.Group(constants.APIBaseProduct + "/:productId/option")
	{
		publicOptionRoutes.GET("", publicRoutesAuth, m.optionHandler.GetAvailableOptions)
	}

	// Auth middleware for protected routes
	sellerAuth := middleware.SellerAuth()

	protectedOptionRoutes := router.Group(constants.APIBaseProduct + "/:productId/option")
	protectedOptionRoutes.Use(sellerAuth)
	{
		protectedOptionRoutes.POST("", m.optionHandler.CreateOption)
		protectedOptionRoutes.PUT("/:optionId", m.optionHandler.UpdateOption)
		protectedOptionRoutes.DELETE("/:optionId", m.optionHandler.DeleteOption)
		protectedOptionRoutes.PUT("/bulk-update", m.optionHandler.BulkUpdateOptions)

		// Option value routes
		protectedOptionRoutes.POST("/:optionId/value", m.valueHandler.AddOptionValue)
		protectedOptionRoutes.PUT("/:optionId/value/:valueId", m.valueHandler.UpdateOptionValue)
		protectedOptionRoutes.DELETE("/:optionId/value/:valueId", m.valueHandler.DeleteOptionValue)
		protectedOptionRoutes.POST("/:optionId/value/bulk", m.valueHandler.BulkAddOptionValues)
		protectedOptionRoutes.PUT(
			"/:optionId/value/bulk-update",
			m.valueHandler.BulkUpdateOptionValues,
		)
	}
}
