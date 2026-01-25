package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/factory/singleton"
	"ecommerce-be/product/handler"

	"github.com/gin-gonic/gin"
)

// CategoryModule implements the Module interface for category routes
type CategoryModule struct {
	categoryHandler *handler.CategoryHandler
}

// NewCategoryModule creates a new instance of CategoryModule
func NewCategoryModule() *CategoryModule {
	f := singleton.GetInstance()

	return &CategoryModule{
		categoryHandler: f.GetCategoryHandler(),
	}
}

// RegisterRoutes registers all category-related routes
func (m *CategoryModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()
	publicRoutesAuth := middleware.PublicAPIAuth()

	// Category routes - /api/product/category/*
	categoryRoutes := router.Group(constants.APIBaseProduct + "/category")
	{
		// Public routes
		categoryRoutes.GET("", publicRoutesAuth, m.categoryHandler.GetAllCategories)
		categoryRoutes.GET("/:categoryId", publicRoutesAuth, m.categoryHandler.GetCategoryByID)
		categoryRoutes.GET("/by-parent", publicRoutesAuth, m.categoryHandler.GetCategoriesByParent)
		categoryRoutes.GET(
			"/:categoryId/attribute",
			publicRoutesAuth,
			m.categoryHandler.GetAttributesByCategoryIDWithInheritance,
		)

		// Admin routes (protected)
		categoryRoutes.POST("", sellerAuth, m.categoryHandler.CreateCategory)
		categoryRoutes.PUT("/:categoryId", sellerAuth, m.categoryHandler.UpdateCategory)
		categoryRoutes.DELETE("/:categoryId", sellerAuth, m.categoryHandler.DeleteCategory)

		// Link/Unlink attribute routes (protected)
		categoryRoutes.POST(
			"/:categoryId/attribute",
			sellerAuth,
			m.categoryHandler.LinkAttributeToCategory,
		)
		categoryRoutes.DELETE(
			"/:categoryId/attribute/:attributeId",
			sellerAuth,
			m.categoryHandler.UnlinkAttributeFromCategory,
		)
	}
}
