package routes

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/factory/singleton"
	"ecommerce-be/product/handlers"

	"github.com/gin-gonic/gin"
)

// AttributeModule implements the Module interface for attribute routes
type AttributeModule struct {
	attributeHandler *handlers.AttributeHandler
}

// NewAttributeModule creates a new instance of AttributeModule
func NewAttributeModule() *AttributeModule {
	f := singleton.GetInstance()

	return &AttributeModule{
		attributeHandler: f.GetAttributeHandler(),
	}
}

// RegisterRoutes registers all attribute-related routes
func (m *AttributeModule) RegisterRoutes(router *gin.Engine) {
	publicRoutesAuth := middleware.PublicAPIAuth()
	auth := middleware.SellerAuth()

	// Attribute routes
	attributeRoutes := router.Group("/api/attributes")
	{

		attributeRoutes.GET("", publicRoutesAuth, m.attributeHandler.GetAllAttributes)
		attributeRoutes.GET("/:attributeId", publicRoutesAuth, m.attributeHandler.GetAttributeByID)

		attributeRoutes.POST("", auth, m.attributeHandler.CreateAttribute)
		attributeRoutes.PUT("/:attributeId", auth, m.attributeHandler.UpdateAttribute)
		attributeRoutes.DELETE("/:attributeId", auth, m.attributeHandler.DeleteAttribute)
		attributeRoutes.POST(
			"/:categoryId",
			auth,
			m.attributeHandler.CreateCategoryAttributeDefinition,
		)
	}
}
