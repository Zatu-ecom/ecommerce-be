package service

import (
	"context"
	"fmt"

	"ecommerce-be/report/factory"
	"ecommerce-be/report/model"
	"ecommerce-be/report/repository"
	"ecommerce-be/report/util"
)

type ReportService interface {
	GetSummary(
		ctx context.Context,
		filter util.ReportQueryFilter,
	) (*model.ReportSummaryResponse, error)
	GetSalesTrends(
		ctx context.Context,
		filter util.ReportQueryFilter,
	) (*model.ReportTrendsResponse, error)
}

type reportService struct {
	reportRepo         repository.ReportRepository
	summaryBuilder     *factory.SummaryResponseBuilder
	salesTrendsBuilder *factory.SalesTrendsResponseBuilder	
}

func NewReportService(
	reportRepo repository.ReportRepository,
	summaryBuilder *factory.SummaryResponseBuilder,
	salesTrendsBuilder *factory.SalesTrendsResponseBuilder,
) ReportService {
	return &reportService{
		reportRepo:         reportRepo,
		summaryBuilder:     summaryBuilder,
		salesTrendsBuilder: salesTrendsBuilder,
	}
}

func (s *reportService) GetSummary(
	ctx context.Context,
	filter util.ReportQueryFilter,
) (*model.ReportSummaryResponse, error) {
	periods, err := util.CalculatePeriods(filter)
	if err != nil {
		return nil, err
	}

	// 2. Fetch Current Period Metrics
	currMetrics, err := s.reportRepo.GetSummaryMetrics(ctx, &periods.CurrStart, &periods.CurrEnd)
	if err != nil {
		return nil, err
	}

	// 3. (Handled by CalculatePeriods)
	duration := periods.CurrEnd.Sub(periods.CurrStart)

	// 4. Fetch Previous Period Metrics
	prevMetrics, err := s.reportRepo.GetSummaryMetrics(ctx, &periods.PrevStart, &periods.PrevEnd)
	if err != nil {
		return nil, err
	}

	// 5. Compute Trends & Deltas (Now delegated to Builder)
	days := int(duration.Hours() / 24)
	compText := fmt.Sprintf("vs previous %d days", days)

	res := s.summaryBuilder.Build(currMetrics, prevMetrics, compText)
	return res, nil
}

func (s *reportService) GetSalesTrends(
	ctx context.Context,
	filter util.ReportQueryFilter,
) (*model.ReportTrendsResponse, error) {
	periods, err := util.CalculatePeriods(filter)
	if err != nil {
		return nil, err
	}

	var interval string
	duration := periods.CurrEnd.Sub(periods.CurrStart)
	hours := duration.Hours()

	if hours <= 24 {
		interval = "hour"
	} else if hours <= 31*24 {
		interval = "day"
	} else if hours <= 90*24 {
		interval = "week"
	} else if hours <= 365*24 {
		interval = "month"
	} else {
		interval = "quarter"
	}

	metrics, err := s.reportRepo.GetSalesTrendsMetrics(ctx, &periods.CurrStart, &periods.CurrEnd, interval)
	if err != nil {
		return nil, err
	}

	res := s.salesTrendsBuilder.Build(metrics, periods.CurrStart, periods.CurrEnd, interval)
	return res, nil
}
