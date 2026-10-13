package factory

import (
	"fmt"
	"math"

	commonModel "ecommerce-be/common/model"
	"ecommerce-be/report/model"
	"ecommerce-be/report/repository"
)

// reportCurrency is the default presentation currency for global reports.
// The report endpoint is global (no seller scope) until seller-scoped reports
// land; per FR-010 all money presentation uses a consistent currency.
var reportCurrency = commonModel.CurrencyInfo{Code: "USD", Symbol: "$", DecimalDigits: 2}

// ReportCurrency returns the default presentation currency for global reports.
func ReportCurrency() commonModel.CurrencyInfo {
	return reportCurrency
}

type SummaryResponseBuilder struct{}

func NewSummaryResponseBuilder() *SummaryResponseBuilder {
	return &SummaryResponseBuilder{}
}

func (b *SummaryResponseBuilder) Build(
	currMetrics, prevMetrics *repository.SummaryMetrics,
	compText string,
) *model.ReportSummaryResponse {
	var currAOVCents, prevAOVCents int64

	if currMetrics.TotalOrders > 0 {
		currAOVCents = currMetrics.TotalRevenue / int64(currMetrics.TotalOrders)
	}
	if prevMetrics.TotalOrders > 0 {
		prevAOVCents = prevMetrics.TotalRevenue / int64(prevMetrics.TotalOrders)
	}

	currRevenue := commonModel.NewMoney(currMetrics.TotalRevenue, reportCurrency)
	prevRevenue := commonModel.NewMoney(prevMetrics.TotalRevenue, reportCurrency)
	currAOV := commonModel.NewMoney(currAOVCents, reportCurrency)
	prevAOV := commonModel.NewMoney(prevAOVCents, reportCurrency)

	return &model.ReportSummaryResponse{
		TotalRevenue: model.MetricFloat{
			Value:            currRevenue,
			FormattedValue:   currRevenue.Formatted,
			PercentageChange: b.calculatePercentageChange(prevRevenue.Amount, currRevenue.Amount),
			Trend:            b.calculateTrend(prevRevenue.Amount, currRevenue.Amount),
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
			FormattedValue:   currAOV.Formatted,
			PercentageChange: b.calculatePercentageChange(prevAOV.Amount, currAOV.Amount),
			Trend:            b.calculateTrend(prevAOV.Amount, currAOV.Amount),
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
