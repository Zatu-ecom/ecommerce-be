package model

type MetricFloat struct {
	Value            float64 `json:"value"`
	FormattedValue   string  `json:"formatted_value"`
	PercentageChange float64 `json:"percentage_change"`
	Trend            string  `json:"trend"`
	ComparisonText   string  `json:"comparison_text"`
}

type MetricInt struct {
	Value            int     `json:"value"`
	FormattedValue   string  `json:"formatted_value"`
	PercentageChange float64 `json:"percentage_change"`
	Trend            string  `json:"trend"`
	ComparisonText   string  `json:"comparison_text"`
}

type ReportSummaryResponse struct {
	TotalRevenue      MetricFloat `json:"total_revenue"`
	TotalOrders       MetricInt   `json:"total_orders"`
	AverageOrderValue MetricFloat `json:"average_order_value"`
	TotalCustomers    MetricInt   `json:"total_customers"`
}

type ReportTrendsResponse struct {
	Interval        string    `json:"interval"`
	Labels          []string  `json:"labels"`
	RevenueData     []float64 `json:"revenue_data"`
	OrderVolumeData []int     `json:"order_volume_data"`
}

type OrderStatusDistribution struct {
	Status     string  `json:"status"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type ReportOrderDistributionResponse struct {
	Distribution []OrderStatusDistribution `json:"distribution"`
}

type TopSellingProduct struct {
	ProductID        string  `json:"product_id"`
	Name             string  `json:"name"`
	SKU              string  `json:"sku"`
	QuantitySold     int     `json:"quantity_sold"`
	RevenueGenerated float64 `json:"revenue_generated"`
}

type ReportTopSellersResponse struct {
	Products []TopSellingProduct `json:"products"`
}

type ReportCustomerRetentionResponse struct {
	Labels                 []string `json:"labels"`
	NewCustomersData       []int    `json:"new_customers_data"`
	ReturningCustomersData []int    `json:"returning_customers_data"`
}

type PromotionPerformanceItem struct {
	PromotionID      string  `json:"promotion_id"`
	Name             string  `json:"name"`
	TimesUsed        int     `json:"times_used"`
	RevenueGenerated float64 `json:"revenue_generated"`
	DiscountGiven    float64 `json:"discount_given"`
}

type PromotionGlobalMetrics struct {
	TotalDiscountAmount       float64 `json:"total_discount_amount"`
	TotalOrdersWithPromo      int     `json:"total_orders_with_promo"`
	PromoUsageRatePercentage  float64 `json:"promo_usage_rate_percentage"`
}

type ReportPromotionPerformanceResponse struct {
	GlobalMetrics PromotionGlobalMetrics     `json:"global_metrics"`
	TopPromotions []PromotionPerformanceItem `json:"top_promotions"`
}
