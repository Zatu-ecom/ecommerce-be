package routes

import (
	"ecommerce-be/common/db"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/handlers"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/service"

	"github.com/gin-gonic/gin"
)

// CategoryModule implements the Module interface for category routes
type CategoryModule struct {
	categoryHandler *handlers.CategoryHandler
}

// NewCategoryModule creates a new instance of CategoryModule
func NewCategoryModule() *CategoryModule {
	categoryRepo := repositories.NewCategoryRepository(db.GetDB())
	productRepo := repositories.NewProductRepository(db.GetDB())
	attributeRepo := repositories.NewAttributeDefinitionRepository(db.GetDB())
	categoryService := service.NewCategoryService(categoryRepo, productRepo, attributeRepo)

	return &CategoryModule{
		categoryHandler: handlers.NewCategoryHandler(categoryService),
	}
}

// RegisterRoutes registers all category-related routes
func (m *CategoryModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()
	publicRoutesAuth := middleware.PublicAPIAuth()

	// Category routes
	categoryRoutes := router.Group("/api/categories")
	{
		// Public routes
		categoryRoutes.GET("", publicRoutesAuth, m.categoryHandler.GetAllCategories)
		categoryRoutes.GET("/:categoryId", publicRoutesAuth, m.categoryHandler.GetCategoryByID)
		categoryRoutes.GET("/by-parent", publicRoutesAuth, m.categoryHandler.GetCategoriesByParent)
		categoryRoutes.GET(
			"/:categoryId/attributes",
			publicRoutesAuth,
			m.categoryHandler.GetAttributesByCategoryIDWithInheritance,
		)

		// Admin routes (protected)
		categoryRoutes.POST("", sellerAuth, m.categoryHandler.CreateCategory)
		categoryRoutes.PUT("/:categoryId", sellerAuth, m.categoryHandler.UpdateCategory)
		categoryRoutes.DELETE("/:categoryId", sellerAuth, m.categoryHandler.DeleteCategory)
		
		// Link/Unlink attribute routes (protected)
		categoryRoutes.POST("/:categoryId/attributes", sellerAuth, m.categoryHandler.LinkAttributeToCategory)
		categoryRoutes.DELETE("/:categoryId/attributes/:attributeId", sellerAuth, m.categoryHandler.UnlinkAttributeFromCategory)
	}
}
