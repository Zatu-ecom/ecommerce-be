package routes

import (
	"ecommerce-be/common"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product_management/handlers"
	"ecommerce-be/product_management/repositories"
	"ecommerce-be/product_management/service"

	"github.com/gin-gonic/gin"
)

// ProductModule implements the Module interface for product routes
type ProductModule struct {
	productHandler *handlers.ProductHandler
}

// NewProductModule creates a new instance of ProductModule
func NewProductModule() *ProductModule {
	categoryRepo := repositories.NewCategoryRepository(common.GetDB())
	attributeRepo := repositories.NewAttributeDefinitionRepository(common.GetDB())
	productRepo := repositories.NewProductRepository(common.GetDB())

	productService := service.NewProductService(productRepo, categoryRepo, attributeRepo)

	return &ProductModule{
		productHandler: handlers.NewProductHandler(productService),
	}
}

// RegisterRoutes registers all product-related routes
func (m *ProductModule) RegisterRoutes(router *gin.Engine) {
	// Auth middleware for protected routes
	auth := middleware.Auth()

	// Product routes
	productRoutes := router.Group("/api/products")
	{
		// Public routes
		productRoutes.GET("", m.productHandler.GetAllProducts)
		productRoutes.GET("/:productId", m.productHandler.GetProductByID)
		productRoutes.GET("/search", m.productHandler.SearchProducts)
		productRoutes.GET("/filters", m.productHandler.GetProductFilters)
		productRoutes.GET("/:productId/related", m.productHandler.GetRelatedProducts)

		// Admin/Seller routes (protected)
		productRoutes.POST("", auth, m.productHandler.CreateProduct)
		productRoutes.PUT("/:productId", auth, m.productHandler.UpdateProduct)
		productRoutes.DELETE("/:productId", auth, m.productHandler.DeleteProduct)
		productRoutes.PATCH("/:productId/stock", auth, m.productHandler.UpdateProductStock)
	}
}
