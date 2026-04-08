package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/report/factory/singleton"
	"ecommerce-be/report/handler"

	"github.com/gin-gonic/gin"
)

type ReportModule struct {
	reportHandler *handler.ReportHandler
}

func NewReportModule() *ReportModule {
	factory := singleton.GetInstance()
	h := factory.GetReportHandler()
	return &ReportModule{
		reportHandler: h,
	}
}

func (m *ReportModule) RegisterRoutes(router *gin.Engine) {
	adminAuth := middleware.AdminAuth() // Admin auth for reports

	reportRoutes := router.Group(constants.APIBaseReport)
	reportRoutes.Use(adminAuth) // Apply admin auth to all report routes

	{
		reportRoutes.GET("/summary", m.reportHandler.GetSummary)
		reportRoutes.GET("/sales/trends", m.reportHandler.GetSalesTrends)
		reportRoutes.GET("/orders/distribution", m.reportHandler.GetOrderDistribution)
		reportRoutes.GET("/products/top-sellers", m.reportHandler.GetTopSellingProducts)
		reportRoutes.GET("/customers/retention", m.reportHandler.GetCustomerRetention)
		reportRoutes.GET("/promotions/performance", m.reportHandler.GetPromotionPerformance)
	}
}
