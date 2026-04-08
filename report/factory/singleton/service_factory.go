package singleton

import (
	"ecommerce-be/report/factory"
	"ecommerce-be/report/service"
)

type ServiceFactory struct {
	reportService service.ReportService
}

func NewServiceFactory(repoFactory *RepositoryFactory) *ServiceFactory {
	summaryBuilder := factory.NewSummaryResponseBuilder()
	trendsBuilder := factory.NewSalesTrendsResponseBuilder()
	return &ServiceFactory{
		reportService: service.NewReportService(
			repoFactory.GetReportRepository(),
			summaryBuilder,
			trendsBuilder,
		),
	}
}

func (f *ServiceFactory) GetReportService() service.ReportService {
	return f.reportService
}
