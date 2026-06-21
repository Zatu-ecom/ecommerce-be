package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/factory/singleton"
	"ecommerce-be/product/handler"

	"github.com/gin-gonic/gin"
)

// CollectionModule implements the Module interface for collection routes
type CollectionModule struct {
	collectionHandler *handler.CollectionHandler
}

// NewCollectionModule creates a new CollectionModule
func NewCollectionModule() *CollectionModule {
	f := singleton.GetInstance()
	return &CollectionModule{
		collectionHandler: f.GetCollectionHandler(),
	}
}

// RegisterRoutes registers all collection-related routes
func (m *CollectionModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()
	publicRoutesAuth := middleware.PublicAPIAuth()

	collectionRoutes := router.Group(constants.APIBaseProduct + "/collection")
	{
		collectionRoutes.GET("", publicRoutesAuth, m.collectionHandler.GetAllCollections)
		collectionRoutes.GET("/:collectionId", publicRoutesAuth, m.collectionHandler.GetCollectionByID)
		collectionRoutes.GET(
			"/:collectionId/product",
			publicRoutesAuth,
			m.collectionHandler.GetProducts,
		)

		collectionRoutes.POST("", sellerAuth, m.collectionHandler.CreateCollection)
		collectionRoutes.PUT("/:collectionId", sellerAuth, m.collectionHandler.UpdateCollection)
		collectionRoutes.DELETE("/:collectionId", sellerAuth, m.collectionHandler.DeleteCollection)

		collectionRoutes.POST(
			"/:collectionId/product",
			sellerAuth,
			m.collectionHandler.AddProducts,
		)
		collectionRoutes.DELETE(
			"/:collectionId/product",
			sellerAuth,
			m.collectionHandler.RemoveProducts,
		)
		collectionRoutes.PUT(
			"/:collectionId/product/reorder",
			sellerAuth,
			m.collectionHandler.ReorderProducts,
		)
	}
}
