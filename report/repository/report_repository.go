package repository

import (
	"context"
	"fmt"
	"time"

	"ecommerce-be/order/entity"

	"gorm.io/gorm"
)

// sanitizeTimezone returns a known-safe IANA timezone name. Falls back to UTC
// for anything that fails the whitelist of characters allowed in IANA names
// (`A-Z a-z 0-9 _ + - /`). Prevents SQL injection via the `AT TIME ZONE`
// clause where a bind parameter cannot be used.
func sanitizeTimezone(tz string) string {
	if tz == "" {
		return "UTC"
	}
	for _, r := range tz {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '+' || r == '-' || r == '/' || r == ':':
		default:
			return "UTC"
		}
	}
	// Numeric offsets like "+05:30" are valid for Postgres `AT TIME ZONE`
	// but not for Go's time.LoadLocation, so only run the Go check when the
	// value looks like an IANA name (contains "/" or equals a bare "UTC"/
	// region-less zone).
	if tz == "UTC" {
		return tz
	}
	hasSlash := false
	for _, r := range tz {
		if r == '/' {
			hasSlash = true
			break
		}
	}
	if hasSlash {
		if _, err := time.LoadLocation(tz); err != nil {
			return "UTC"
		}
	}
	return tz
}

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
		timezone string,
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
	timezone string,
) ([]TrendMetric, error) {
	var metrics []TrendMetric

	validStatuses := []string{
		string(entity.ORDER_STATUS_CONFIRMED),
		string(entity.ORDER_STATUS_COMPLETED),
	}

	tz := sanitizeTimezone(timezone)

	// Storage is UTC (timestamptz) but presentation buckets must align with
	// the caller's timezone — otherwise an IST client sees UTC-offset hours
	// and UTC-offset days. `AT TIME ZONE '<tz>'` converts the timestamptz
	// into a naive timestamp in that zone before DATE_TRUNC buckets it.
	// tz is sanitised above so embedding it is safe; also interval is an
	// internal enum (not user input).
	bucketExpr := fmt.Sprintf(
		"DATE_TRUNC('%s', placed_at AT TIME ZONE '%s')",
		interval,
		tz,
	)

	selectQuery := fmt.Sprintf(
		"TO_CHAR(%s, 'YYYY-MM-DD HH24:MI:SS') as date, "+
			"COALESCE(SUM(total_cents)/100.0, 0) as total_revenue, "+
			"COUNT(id) as total_orders",
		bucketExpr,
	)

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

	query = query.Group(bucketExpr).Order(bucketExpr + " ASC")

	if err := query.Scan(&metrics).Error; err != nil {
		return nil, err
	}

	return metrics, nil
}
