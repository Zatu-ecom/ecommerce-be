package repository

import (
	"context"
	"fmt"
	"time"

	"ecommerce-be/order/entity"

	"gorm.io/gorm"
)

type SummaryMetrics struct {
	TotalRevenue   int64 `gorm:"column:total_revenue"`
	TotalOrders    int   `gorm:"column:total_orders"`
	TotalCustomers int   `gorm:"column:total_customers"`
}

type TrendMetric struct {
	Date         string  `gorm:"column:date"`
	TotalRevenue float64 `gorm:"column:total_revenue"`
	TotalOrders  int     `gorm:"column:total_orders"`
}

type ReportRepository interface {
	GetSummaryMetrics(
		ctx context.Context,
		startDate, endDate *time.Time,
	) (*SummaryMetrics, error)
	GetSalesTrendsMetrics(
		ctx context.Context,
		startDate, endDate *time.Time,
		interval string,
	) ([]TrendMetric, error)
}

type reportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) ReportRepository {
	return &reportRepository{
		db: db,
	}
}

func (r *reportRepository) GetSummaryMetrics(
	ctx context.Context,
	startDate, endDate *time.Time,
) (*SummaryMetrics, error) {
	var metrics SummaryMetrics

	validStatuses := []string{
		string(entity.ORDER_STATUS_CONFIRMED),
		string(entity.ORDER_STATUS_COMPLETED),
	}

	query := r.db.WithContext(ctx).
		Model(&entity.Order{}).
		Select(`
			COALESCE(SUM(total_cents), 0) as total_revenue,
			COUNT(id) as total_orders,
			COUNT(DISTINCT user_id) as total_customers
		`).
		Where("status IN ?", validStatuses)

	if startDate != nil {
		query = query.Where("placed_at >= ?", startDate)
	}
	if endDate != nil {
		query = query.Where("placed_at <= ?", endDate)
	}

	err := query.Scan(&metrics).Error
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}

func (r *reportRepository) GetSalesTrendsMetrics(
	ctx context.Context,
	startDate, endDate *time.Time,
	interval string,
) ([]TrendMetric, error) {
	var metrics []TrendMetric

	validStatuses := []string{
		string(entity.ORDER_STATUS_CONFIRMED),
		string(entity.ORDER_STATUS_COMPLETED),
	}

	// We use Postgres DATE_TRUNC to easily bucket timestamps.
	// The interval parameter maps nicely to 'hour', 'day', 'month', etc.
	selectQuery := fmt.Sprintf(`
		TO_CHAR(DATE_TRUNC('%s', placed_at), 'YYYY-MM-DD HH24:MI:SS') as date,
		COALESCE(SUM(total_cents)/100.0, 0) as total_revenue,
		COUNT(id) as total_orders
	`, interval)

	query := r.db.WithContext(ctx).
		Model(&entity.Order{}).
		Select(selectQuery).
		Where("status IN ?", validStatuses)

	if startDate != nil {
		query = query.Where("placed_at >= ?", startDate)
	}
	if endDate != nil {
		query = query.Where("placed_at <= ?", endDate)
	}

	groupStr := fmt.Sprintf("DATE_TRUNC('%s', placed_at)", interval)
	query = query.Group(groupStr).Order(groupStr + " ASC")

	if err := query.Scan(&metrics).Error; err != nil {
		return nil, err
	}

	return metrics, nil
}
