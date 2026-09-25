package route

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/promotion/factory/singleton"
	"ecommerce-be/promotion/handler"
	"ecommerce-be/promotion/utils/constant"

	"github.com/gin-gonic/gin"
)

// DiscountCodeModule implements the Module interface for discount-code routes.
type DiscountCodeModule struct {
	discountCodeHandler *handler.DiscountCodeHandler
}

// NewDiscountCodeModule creates a new DiscountCodeModule.
func NewDiscountCodeModule() *DiscountCodeModule {
	f := singleton.GetInstance()
	return &DiscountCodeModule{
		discountCodeHandler: f.GetDiscountCodeHandler(),
	}
}

// RegisterRoutes registers all seller discount-code routes.
func (m *DiscountCodeModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	routes := router.Group(constant.API_BASE_DISCOUNT_CODE)
	{
		routes.POST("", sellerAuth, m.discountCodeHandler.CreateDiscountCode)
		routes.GET("", sellerAuth, m.discountCodeHandler.ListDiscountCodes)
		routes.GET("/:id", sellerAuth, m.discountCodeHandler.GetDiscountCode)
		routes.PUT("/:id", sellerAuth, m.discountCodeHandler.UpdateDiscountCode)
		routes.DELETE("/:id", sellerAuth, m.discountCodeHandler.DeleteDiscountCode)
		routes.PATCH("/:id/status", sellerAuth, m.discountCodeHandler.UpdateStatus)
	}
}
