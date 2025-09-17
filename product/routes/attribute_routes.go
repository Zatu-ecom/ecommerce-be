package routes

import (
	"ecommerce-be/common/db"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/handlers"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/service"

	"github.com/gin-gonic/gin"
)

// AttributeModule implements the Module interface for attribute routes
type AttributeModule struct {
	attributeHandler *handlers.AttributeHandler
}

// NewAttributeModule creates a new instance of AttributeModule
func NewAttributeModule() *AttributeModule {
	attributeRepo := repositories.NewAttributeDefinitionRepository(db.GetDB())
	attributeService := service.NewAttributeDefinitionService(attributeRepo)

	return &AttributeModule{
		attributeHandler: handlers.NewAttributeHandler(attributeService),
	}
}

// RegisterRoutes registers all attribute-related routes
func (m *AttributeModule) RegisterRoutes(router *gin.Engine) {
	// Auth middleware for protected routes
	auth := middleware.Auth()

	// Attribute routes
	attributeRoutes := router.Group("/api/attributes")
	{
		// Public routes
		attributeRoutes.GET("", m.attributeHandler.GetAllAttributes)
		attributeRoutes.GET("/:attributeId", m.attributeHandler.GetAttributeByID)

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
