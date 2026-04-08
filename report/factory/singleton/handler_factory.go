package singleton

import (
	"ecommerce-be/report/handler"
)

type HandlerFactory struct {
	reportHandler *handler.ReportHandler
}

func NewHandlerFactory(serviceFactory *ServiceFactory) *HandlerFactory {
	return &HandlerFactory{
		reportHandler: handler.NewReportHandler(
			serviceFactory.GetReportService(),
		),
	}
}

func (f *HandlerFactory) GetReportHandler() *handler.ReportHandler {
	return f.reportHandler
}
