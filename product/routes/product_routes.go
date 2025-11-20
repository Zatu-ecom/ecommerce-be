package routes

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/factory/singleton"
	"ecommerce-be/product/handlers"

	"github.com/gin-gonic/gin"
)

// ProductModule implements the Module interface for product routes
type ProductModule struct {
	productHandler *handlers.ProductHandler
}

// NewProductModule creates a new instance of ProductModule
func NewProductModule() *ProductModule {
	f := singleton.GetInstance()

	return &ProductModule{
		productHandler: f.GetProductHandler(),
	}
}

// RegisterRoutes registers all product-related routes
func (m *ProductModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()
	publicRoutesAuth := middleware.PublicAPIAuth()

	// Product routes
	productRoutes := router.Group("/api/products")
	{
		// Public routes
		productRoutes.GET("", publicRoutesAuth, m.productHandler.GetAllProducts)
		productRoutes.GET("/:productId", publicRoutesAuth, m.productHandler.GetProductByID)
		productRoutes.GET("/search", publicRoutesAuth, m.productHandler.SearchProducts)
		productRoutes.GET("/filters", publicRoutesAuth, m.productHandler.GetProductFilters)
		productRoutes.GET(
			"/:productId/related",
			publicRoutesAuth,
			m.productHandler.GetRelatedProductsScored,
		)

		// Admin/Seller routes (protected)
		productRoutes.POST("", sellerAuth, m.productHandler.CreateProduct)
		productRoutes.PUT("/:productId", sellerAuth, m.productHandler.UpdateProduct)
		productRoutes.DELETE("/:productId", sellerAuth, m.productHandler.DeleteProduct)
	}
}
