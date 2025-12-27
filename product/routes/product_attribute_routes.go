package routes

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/factory/singleton"
	"ecommerce-be/product/handlers"

	"github.com/gin-gonic/gin"
)

// ProductAttributeModule implements the Module interface for product attribute routes
type ProductAttributeModule struct {
	productAttrHandler *handlers.ProductAttributeHandler
}

// NewProductAttributeModule creates a new instance of ProductAttributeModule
func NewProductAttributeModule() *ProductAttributeModule {
	f := singleton.GetInstance()

	return &ProductAttributeModule{
		productAttrHandler: f.GetProductAttributeHandler(),
	}
}

// RegisterRoutes registers all product attribute-related routes
func (m *ProductAttributeModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()
	publicRoutesAuth := middleware.PublicAPIAuth()

	// Product Attribute routes - nested under products - /api/product/products/:productId/attributes/*
	productAttrRoutes := router.Group(constants.APIBaseProduct + "/products/:productId/attributes")
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
