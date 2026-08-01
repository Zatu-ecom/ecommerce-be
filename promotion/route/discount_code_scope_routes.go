package route

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/promotion/factory/singleton"
	"ecommerce-be/promotion/handler"
	"ecommerce-be/promotion/utils/constant"

	"github.com/gin-gonic/gin"
)

// DiscountCodeScopeModule registers discount-code catalog scope routes.
type DiscountCodeScopeModule struct {
	productHandler    *handler.DiscountCodeProductScopeHandler
	variantHandler    *handler.DiscountCodeVariantScopeHandler
	categoryHandler   *handler.DiscountCodeCategoryScopeHandler
	collectionHandler *handler.DiscountCodeCollectionScopeHandler
}

// NewDiscountCodeScopeModule creates a new DiscountCodeScopeModule.
func NewDiscountCodeScopeModule() *DiscountCodeScopeModule {
	f := singleton.GetInstance()
	return &DiscountCodeScopeModule{
		productHandler:    f.GetDiscountCodeProductScopeHandler(),
		variantHandler:    f.GetDiscountCodeVariantScopeHandler(),
		categoryHandler:   f.GetDiscountCodeCategoryScopeHandler(),
		collectionHandler: f.GetDiscountCodeCollectionScopeHandler(),
	}
}

// RegisterRoutes registers all discount-code scope routes.
func (m *DiscountCodeScopeModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	routes := router.Group(constant.API_BASE_DISCOUNT_CODE + "/scope")
	{
		routes.POST("/product", sellerAuth, m.productHandler.AddProducts)
		routes.DELETE("/product", sellerAuth, m.productHandler.RemoveProducts)
		routes.DELETE("/:discountCodeId/product", sellerAuth, m.productHandler.RemoveAllProducts)
		routes.GET("/:discountCodeId/product", sellerAuth, m.productHandler.GetProducts)

		routes.POST("/variant", sellerAuth, m.variantHandler.AddVariants)
		routes.DELETE("/variant", sellerAuth, m.variantHandler.RemoveVariants)
		routes.DELETE("/:discountCodeId/variant", sellerAuth, m.variantHandler.RemoveAllVariants)
		routes.GET("/:discountCodeId/variant", sellerAuth, m.variantHandler.GetVariants)

		routes.POST("/category", sellerAuth, m.categoryHandler.AddCategories)
		routes.DELETE("/category", sellerAuth, m.categoryHandler.RemoveCategories)
		routes.DELETE(
			"/:discountCodeId/category",
			sellerAuth,
			m.categoryHandler.RemoveAllCategories,
		)
		routes.GET("/:discountCodeId/category", sellerAuth, m.categoryHandler.GetCategories)

		routes.POST("/collection", sellerAuth, m.collectionHandler.AddCollections)
		routes.DELETE("/collection", sellerAuth, m.collectionHandler.RemoveCollections)
		routes.DELETE(
			"/:discountCodeId/collection",
			sellerAuth,
			m.collectionHandler.RemoveAllCollections,
		)
		routes.GET("/:discountCodeId/collection", sellerAuth, m.collectionHandler.GetCollections)
	}
}
