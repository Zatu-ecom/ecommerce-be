package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/factory/singleton"
	"ecommerce-be/product/handler"

	"github.com/gin-gonic/gin"
)

// PackageOptionModule implements the Module interface for package option routes
type PackageOptionModule struct {
	packageOptionHandler *handler.PackageOptionHandler
}

// NewPackageOptionModule creates a new instance of PackageOptionModule
func NewPackageOptionModule() *PackageOptionModule {
	f := singleton.GetInstance()

	return &PackageOptionModule{
		packageOptionHandler: f.GetPackageOptionHandler(),
	}
}

// RegisterRoutes registers all package option-related routes
func (m *PackageOptionModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()
	publicRoutesAuth := middleware.PublicAPIAuth()

	packageOptionRoutes := router.Group(constants.APIBaseProduct + "/:productId/package-option")
	{
		packageOptionRoutes.GET("", publicRoutesAuth, m.packageOptionHandler.GetPackageOptions)

		packageOptionRoutes.POST("", sellerAuth, m.packageOptionHandler.AddPackageOption)
		packageOptionRoutes.PUT("/bulk", sellerAuth, m.packageOptionHandler.BulkUpdatePackageOptions)
		packageOptionRoutes.PUT(
			"/:packageOptionId",
			sellerAuth,
			m.packageOptionHandler.UpdatePackageOption,
		)
		packageOptionRoutes.DELETE(
			"/:packageOptionId",
			sellerAuth,
			m.packageOptionHandler.DeletePackageOption,
		)
	}
}
