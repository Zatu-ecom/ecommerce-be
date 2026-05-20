package factory

import (
	"fmt"
	"time"

	"ecommerce-be/report/model"
	"ecommerce-be/report/repository"
)

type SalesTrendsResponseBuilder struct{}

func NewSalesTrendsResponseBuilder() *SalesTrendsResponseBuilder {
	return &SalesTrendsResponseBuilder{}
}

func (b *SalesTrendsResponseBuilder) Build(
	metrics []repository.TrendMetric,
	startDate time.Time,
	endDate time.Time,
	interval string,
) *model.ReportTrendsResponse {
	// Create lookup mechanism
	metricsMap := make(map[string]repository.TrendMetric)
	for _, m := range metrics {
		metricsMap[m.Date] = m
	}

	res := &model.ReportTrendsResponse{
		Interval:        interval,
		Labels:          []string{},
		RevenueData:     []float64{},
		OrderVolumeData: []int{},
	}

	// Calculate points array size based on loop iterations
	current := startDate

	// Determine formatting step and increment logic
	var increment func(time.Time) time.Time
	var format string

	switch interval {
	case "hour":
		format = "15:00"
		increment = func(t time.Time) time.Time { return t.Add(1 * time.Hour) }
	case "day":
		format = "2006-01-02"
		increment = func(t time.Time) time.Time { return t.AddDate(0, 0, 1) }
	case "month":
		format = "Jan 2006"
		// Set to start of month for clean increments
		current = time.Date(
			startDate.Year(),
			startDate.Month(),
			1,
			0,
			0,
			0,
			0,
			startDate.Location(),
		)
		increment = func(t time.Time) time.Time { return t.AddDate(0, 1, 0) }
	case "week":
		format = "2006-01-02"
		increment = func(t time.Time) time.Time { return t.AddDate(0, 0, 7) }
	case "quarter":
		format = "2006 Q1" // Custom logic applied later
		increment = func(t time.Time) time.Time { return t.AddDate(0, 3, 0) }
	case "year":
		format = "2006"
		increment = func(t time.Time) time.Time { return t.AddDate(1, 0, 0) }
	default:
		format = "2006-01-02"
		increment = func(t time.Time) time.Time { return t.AddDate(0, 0, 1) }
	}

	for current.Before(endDate) || current.Equal(endDate) {
		var dbKey string
		var label string

		// The repository now truncates `placed_at AT TIME ZONE <caller TZ>`
		// before formatting, so the DB emits keys in the caller's location.
		// `current` is iterated in that same location, so plain formatting
		// produces matching keys.
		dbFormat := "2006-01-02 15:04:05"
		dbKey = current.Format(dbFormat)

		if interval == "quarter" {
			q := (current.Month()-1)/3 + 1
			label = fmt.Sprintf("%d Q%d", current.Year(), q)
		} else {
			label = current.Format(format)
		}

		res.Labels = append(res.Labels, label)

		if data, ok := metricsMap[dbKey]; ok {
			res.RevenueData = append(res.RevenueData, data.TotalRevenue)
			res.OrderVolumeData = append(res.OrderVolumeData, data.TotalOrders)
		} else {
			res.RevenueData = append(res.RevenueData, 0.0)
			res.OrderVolumeData = append(res.OrderVolumeData, 0)
		}

		current = increment(current)

		// safeguard against infinite loops, especially for months/years rounding
		// if endpoint is somehow far in future. Stop when array hits arbitrary large
		if len(res.Labels) > 1000 {
			break
		}
	}

	return res
}
