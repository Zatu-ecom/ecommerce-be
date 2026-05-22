# Reports & Dashboard API Design

## Overview
This document outlines the APIs required to power the e-commerce admin dashboard. The API endpoints follow RESTful conventions.

## 1. Universal Time Filtering
All report API endpoints share a common time-filtering query parameter schema to support varied date ranges, relative offsets, and absolute ranges.

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `time_range` | string | Yes | The type of range. Enum: `custom`, `today`, `yesterday`, `this_week`, `this_month`, `this_quarter`, `this_year`, `last_x` |
| `last_x_amount` | int | Conditional | Required if `time_range=last_x`. E.g., `2` |
| `last_x_unit` | string | Conditional | Required if `time_range=last_x`. Enum: `hour`, `day`, `week`, `month`, `quarter`, `year` |
| `start_date` | string | Conditional | Required if `time_range=custom`. ISO8601 UTC format. |
| `end_date` | string | Conditional | Required if `time_range=custom`. ISO8601 UTC format. |
| `interval` | string | No | Granularity for chart data. Enum: `hour`, `day`, `week`, `month`, `year`. Defaults to `day`. |
| `compare` | boolean| No | If `true`, returns the percentage change compared to the identical preceding period. Default `true` for summary endpoints. |

**Example Requests:**
- Last 2 months: `?time_range=last_x&last_x_amount=2&last_x_unit=month`
- Today: `?time_range=today`
- Custom range: `?time_range=custom&start_date=2024-01-01T00:00:00Z&end_date=2024-01-31T23:59:59Z`

---

## 2. API Endpoints

### 2.1 Dashboard Summary Metrics (KPI Cards)
Returns high-level summary metrics for Revenue, Orders, Average Order Value (AOV), and Customers.

**Endpoint:** `GET /admin/api/v1/reports/summary`

**Request:** `GET /admin/api/v1/reports/summary?time_range=last_x&last_x_amount=30&last_x_unit=day&compare=true`

**Response:**
```json
{
  "total_revenue": {
    "value": 45231.89,
    "formatted_value": "$45,231.89",
    "percentage_change": 20.1,
    "trend": "up",
    "comparison_text": "vs previous 30 days"
  },
  "total_orders": {
    "value": 2350,
    "formatted_value": "2,350",
    "percentage_change": 180.1,
    "trend": "up",
    "comparison_text": "vs previous 30 days"
  },
  "average_order_value": {
    "value": 19.24,
    "formatted_value": "$19.24",
    "percentage_change": -2.5,
    "trend": "down",
    "comparison_text": "vs previous 30 days"
  },
  "total_customers": {
    "value": 12234,
    "formatted_value": "12,234",
    "percentage_change": 19.0,
    "trend": "up",
    "comparison_text": "vs previous 30 days"
  }
}
```

### 2.2 Sales & Order Trends (Line/Bar Charts)
Returns time-series data grouped by the requested `interval` to populate line and bar charts.

**Endpoint:** `GET /admin/api/v1/reports/sales/trends`

**Request:** `GET /admin/api/v1/reports/sales/trends?time_range=this_year&interval=month`

**Response:**
```json
{
  "interval": "month",
  "labels": ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"],
  "revenue_data": [4000.0, 2500.0, 2000.0, 2800.0, 1800.0, 2200.0, 3500.0, 0, 0, 0, 0, 0],
  "order_volume_data": [2400, 1300, 10000, 3900, 4800, 3800, 4300, 0, 0, 0, 0, 0]
}
```

### 2.3 Order Status Distribution (Donut Chart)
Shows the breakdown of orders by their current fulfillment/processing status.

**Endpoint:** `GET /admin/api/v1/reports/orders/distribution`

**Request:** `GET /admin/api/v1/reports/orders/distribution?time_range=last_x&last_x_amount=7&last_x_unit=day`

**Response:**
```json
{
  "distribution": [
    {
      "status": "pending",
      "count": 150,
      "percentage": 10.5
    },
    {
      "status": "processing",
      "count": 300,
      "percentage": 21.0
    },
    {
      "status": "shipped",
      "count": 950,
      "percentage": 66.4
    },
    {
      "status": "delivered",
      "count": 80,
      "percentage": 5.0
    },
    {
      "status": "cancelled",
      "count": 30,
      "percentage": 2.1
    }
  ]
}
```

### 2.4 Top Selling Products (Table)
Returns a list of the most successful products in the given timeframe based on revenue or quantity.

**Endpoint:** `GET /admin/api/v1/reports/products/top-sellers`

**Query Additions:**
- `limit` (default 10)
- `sort_by` (`revenue` or `quantity`, default `revenue`)

**Request:** `GET /admin/api/v1/reports/products/top-sellers?time_range=this_month&limit=5&sort_by=revenue`

**Response:**
```json
{
  "products": [
    {
      "product_id": "prod_8374",
      "name": "Wireless Noise Cancelling Headphones",
      "sku": "HEAD-WNC-01",
      "quantity_sold": 345,
      "revenue_generated": 85905.00
    },
    {
      "product_id": "prod_1029",
      "name": "Mechanical Gaming Keyboard",
      "sku": "KB-MECH-02",
      "quantity_sold": 512,
      "revenue_generated": 66508.80
    }
  ]
}
```

### 2.5 Customer Retention (Stacked Bar Chart)
Shows the split between First-Time Buyers and Returning Customers over time.

**Endpoint:** `GET /admin/api/v1/reports/customers/retention`

**Request:** `GET /admin/api/v1/reports/customers/retention?time_range=last_x&last_x_amount=6&last_x_unit=month&interval=month`

**Response:**
```json
{
  "labels": ["Feb", "Mar", "Apr", "May", "Jun", "Jul"],
  "new_customers_data": [400, 350, 410, 390, 420, 500],
  "returning_customers_data": [600, 750, 780, 810, 850, 920]
}
```

### 2.6 Promotion & Discount Performance (Cards/Table)
Shows how efficiently discounts and promo codes are driving sales.

**Endpoint:** `GET /admin/api/v1/reports/promotions/performance`

**Request:** `GET /admin/api/v1/reports/promotions/performance?time_range=this_year`

**Response:**
```json
{
  "global_metrics": {
    "total_discount_amount": 12500.50,
    "total_orders_with_promo": 850,
    "promo_usage_rate_percentage": 15.4 
  },
  "top_promotions": [
    {
      "promotion_id": "promo_summer_50",
      "name": "Summer 50% Off",
      "times_used": 420,
      "revenue_generated": 45000.00,
      "discount_given": 45000.00
    },
    {
      "promotion_id": "promo_buy1_get1",
      "name": "BOGO Shoes",
      "times_used": 210,
      "revenue_generated": 21000.00,
      "discount_given": 10500.00
    }
  ]
}
```

## 3. Implementation Notes

### Handling the "Compare with Previous" Logic
When `compare=true` is provided, the backend must calculate the precise offset of the time range to query the database a second time.
- If `time_range=last_x` & `last_x_amount=30` & `last_x_unit=day` (Last 30 days): The previous period is exactly the 30 days prior to the start of the current period.
- If `time_range=this_month`: The previous period is `last_month`.
- Formally: `Percentage Change = ((Current - Previous) / Previous) * 100`

### Database Optimizations
- Generating exact charts and metrics over millions of rows on demand can hit the database hard.
- **Recommendation**: Ensure indexes exist on `created_at` (or `order_date`) columns.
- Consider utilizing Materialized Views or separate Analytics tables/cron jobs to build aggregate data nightly for historical reporting, while only hitting the live transactional tables for "today/yesterday/last X hours" fast queries.
