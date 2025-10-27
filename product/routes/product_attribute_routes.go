package routes

import (
	"ecommerce-be/common/db"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/handlers"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/service"

	"github.com/gin-gonic/gin"
)

// ProductAttributeModule implements the Module interface for product attribute routes
type ProductAttributeModule struct {
	productAttrHandler *handlers.ProductAttributeHandler
}

// NewProductAttributeModule creates a new instance of ProductAttributeModule
func NewProductAttributeModule() *ProductAttributeModule {
	productAttrRepo := repositories.NewProductAttributeRepository(db.GetDB())
	productRepo := repositories.NewProductRepository(db.GetDB())
	attributeRepo := repositories.NewAttributeDefinitionRepository(db.GetDB())

	productAttrService := service.NewProductAttributeService(
		productAttrRepo,
		productRepo,
		attributeRepo,
	)

	return &ProductAttributeModule{
		productAttrHandler: handlers.NewProductAttributeHandler(productAttrService),
	}
}

// RegisterRoutes registers all product attribute-related routes
func (m *ProductAttributeModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()
	publicRoutesAuth := middleware.PublicAPIAuth()

	// Product Attribute routes - nested under products
	productAttrRoutes := router.Group("/api/products/:productId/attributes")
	{
		// Public route - get product attributes
		productAttrRoutes.GET("", publicRoutesAuth, m.productAttrHandler.GetProductAttributes)

		// Protected routes - seller/admin only
		productAttrRoutes.POST("", sellerAuth, m.productAttrHandler.AddProductAttribute)
		productAttrRoutes.PUT("/bulk", sellerAuth, m.productAttrHandler.BulkUpdateProductAttributes)
		productAttrRoutes.PUT(
			"/:attributeId",
			sellerAuth,
			m.productAttrHandler.UpdateProductAttribute,
		)
		productAttrRoutes.DELETE(
			"/:attributeId",
			sellerAuth,
			m.productAttrHandler.DeleteProductAttribute,
		)
	}
}
