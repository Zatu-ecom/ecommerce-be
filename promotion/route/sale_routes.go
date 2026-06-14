package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/promotion/factory/singleton"
	"ecommerce-be/promotion/handler"

	"github.com/gin-gonic/gin"
)

// SaleModule implements the Module interface for sale routes
type SaleModule struct {
	saleHandler *handler.SaleHandler
}

// NewSaleModule creates a new SaleModule
func NewSaleModule() *SaleModule {
	f := singleton.GetInstance()
	return &SaleModule{
		saleHandler: f.GetSaleHandler(),
	}
}

// RegisterRoutes registers all sale-related routes
func (m *SaleModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	saleRoutes := router.Group(constants.APIBasePromotion + "/sale")
	{
		saleRoutes.POST("", sellerAuth, m.saleHandler.CreateSale)
		saleRoutes.GET("", sellerAuth, m.saleHandler.ListSales)
		saleRoutes.GET("/:saleId", sellerAuth, m.saleHandler.GetSale)
		saleRoutes.PUT("/:saleId", sellerAuth, m.saleHandler.UpdateSale)
		saleRoutes.DELETE("/:saleId", sellerAuth, m.saleHandler.DeleteSale)
		saleRoutes.PATCH("/:saleId/status", sellerAuth, m.saleHandler.UpdateStatus)
	}
}
