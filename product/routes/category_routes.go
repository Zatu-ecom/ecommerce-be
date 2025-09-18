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
	categoryService := service.NewCategoryService(categoryRepo)

	return &CategoryModule{
		categoryHandler: handlers.NewCategoryHandler(categoryService),
	}
}

// RegisterRoutes registers all category-related routes
func (m *CategoryModule) RegisterRoutes(router *gin.Engine) {
	// Auth middleware for protected routes
	auth := middleware.SellerAuth()

	// Category routes
	categoryRoutes := router.Group("/api/categories")
	{
		// Public routes
		categoryRoutes.GET("", m.categoryHandler.GetAllCategories)
		categoryRoutes.GET("/:categoryId", m.categoryHandler.GetCategoryByID)
		categoryRoutes.GET("/by-parent", m.categoryHandler.GetCategoriesByParent)

		// Admin routes (protected)
		categoryRoutes.POST("", auth, m.categoryHandler.CreateCategory)
		categoryRoutes.PUT("/:categoryId", auth, m.categoryHandler.UpdateCategory)
		categoryRoutes.DELETE("/:categoryId", auth, m.categoryHandler.DeleteCategory)
		categoryRoutes.GET(
			"/:categoryId/attributes",
			auth,
			m.categoryHandler.GetAttributesByCategoryIDWithInheritance,
		)
	}
}
