package factory

import (
	"fmt"
	"math"

	"ecommerce-be/report/model"
	"ecommerce-be/report/repository"
)

type SummaryResponseBuilder struct{}

func NewSummaryResponseBuilder() *SummaryResponseBuilder {
	return &SummaryResponseBuilder{}
}

func (b *SummaryResponseBuilder) Build(
	currMetrics, prevMetrics *repository.SummaryMetrics,
	compText string,
) *model.ReportSummaryResponse {
	var currAOV, prevAOV float64

	if currMetrics.TotalOrders > 0 {
		currAOV = float64(currMetrics.TotalRevenue) / float64(currMetrics.TotalOrders) / 100.0
	}
	if prevMetrics.TotalOrders > 0 {
		prevAOV = float64(prevMetrics.TotalRevenue) / float64(prevMetrics.TotalOrders) / 100.0
	}

	totalRevenueFloat := float64(currMetrics.TotalRevenue) / 100.0
	prevRevenueFloat := float64(prevMetrics.TotalRevenue) / 100.0

	return &model.ReportSummaryResponse{
		TotalRevenue: model.MetricFloat{
			Value:            totalRevenueFloat,
			FormattedValue:   fmt.Sprintf("$%.2f", totalRevenueFloat),
			PercentageChange: b.calculatePercentageChange(prevRevenueFloat, totalRevenueFloat),
			Trend:            b.calculateTrend(prevRevenueFloat, totalRevenueFloat),
			ComparisonText:   compText,
		},
		TotalOrders: model.MetricInt{
			Value:          currMetrics.TotalOrders,
			FormattedValue: fmt.Sprintf("%d", currMetrics.TotalOrders),
			PercentageChange: b.calculatePercentageChange(
				float64(prevMetrics.TotalOrders),
				float64(currMetrics.TotalOrders),
			),
			Trend: b.calculateTrend(
				float64(prevMetrics.TotalOrders),
				float64(currMetrics.TotalOrders),
			),
			ComparisonText: compText,
		},
		AverageOrderValue: model.MetricFloat{
			Value:            currAOV,
			FormattedValue:   fmt.Sprintf("$%.2f", currAOV),
			PercentageChange: b.calculatePercentageChange(prevAOV, currAOV),
			Trend:            b.calculateTrend(prevAOV, currAOV),
			ComparisonText:   compText,
		},
		TotalCustomers: model.MetricInt{
			Value:          currMetrics.TotalCustomers,
			FormattedValue: fmt.Sprintf("%d", currMetrics.TotalCustomers),
			PercentageChange: b.calculatePercentageChange(
				float64(prevMetrics.TotalCustomers),
				float64(currMetrics.TotalCustomers),
			),
			Trend: b.calculateTrend(
				float64(prevMetrics.TotalCustomers),
				float64(currMetrics.TotalCustomers),
			),
			ComparisonText: compText,
		},
	}
}

func (b *SummaryResponseBuilder) calculatePercentageChange(prev, curr float64) float64 {
	if prev == 0 {
		if curr == 0 {
			return 0.0
		}
		return 100.0
	}

	change := ((curr - prev) / prev) * 100
	return math.Round(change*100) / 100
}

func (b *SummaryResponseBuilder) calculateTrend(prev, curr float64) string {
	if curr > prev {
		return "up"
	}
	if curr < prev {
		return "down"
	}
	return "flat"
}
