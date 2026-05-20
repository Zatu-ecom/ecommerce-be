package report_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	orderEntity "ecommerce-be/order/entity"
	reportModel "ecommerce-be/report/model"
	"ecommerce-be/test/integration/helpers"
)

func (s *ReportSuite) TestGetSummaryReport_TrendFlatAndDown() {
	s.cleanupDomainData() // explicitly clean before individual scenario
	now := time.Now()
	thirtyFiveDaysAgo := now.AddDate(0, 0, -35)

	orders := []orderEntity.Order{
		// Current Period (150.00, 2 orders, 1 customer)
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-1",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  10000,
			PlacedAt:    &now,
		},
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-2",
			Status:      orderEntity.ORDER_STATUS_CONFIRMED,
			TotalCents:  5000,
			PlacedAt:    &now,
		},

		// Previous Period (150.00, 2 orders, 2 customers)
		{
			UserID:      helpers.Seller2UserID,
			OrderNumber: "ORD-3",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  10000,
			PlacedAt:    &thirtyFiveDaysAgo,
		},
		{
			UserID:      helpers.Customer2UserID,
			OrderNumber: "ORD-4",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  5000,
			PlacedAt:    &thirtyFiveDaysAgo,
		},
	}

	for _, o := range orders {
		s.Require().NoError(s.container.DB.Create(&o).Error)
	}

	res := s.adminClient.Get(s.T(), "/api/report/summary")
	s.Require().Equal(http.StatusOK, res.Code)

	var responseBody struct {
		Data reportModel.ReportSummaryResponse `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &responseBody))

	// Assert Revenue (150 vs 150) -> flat
	s.Equal(150.0, responseBody.Data.TotalRevenue.Value)
	s.Equal(0.0, responseBody.Data.TotalRevenue.PercentageChange)
	s.Equal("flat", responseBody.Data.TotalRevenue.Trend)

	// Assert Orders (2 vs 2) -> flat
	s.Equal("flat", responseBody.Data.TotalOrders.Trend)

	// Assert Customers (1 vs 2) -> down (-50%)
	s.Equal(-50.0, responseBody.Data.TotalCustomers.PercentageChange)
	s.Equal("down", responseBody.Data.TotalCustomers.Trend)
}

func (s *ReportSuite) TestGetSummaryReport_TrendUp() {
	s.cleanupDomainData()
	now := time.Now()
	fortyDaysAgo := now.AddDate(0, 0, -40)

	orders := []orderEntity.Order{
		// Current Period (300.00, 3 orders, 2 customers)
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-1",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  20000,
			PlacedAt:    &now,
		},
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-2",
			Status:      orderEntity.ORDER_STATUS_CONFIRMED,
			TotalCents:  5000,
			PlacedAt:    &now,
		},
		{
			UserID:      helpers.Seller2UserID,
			OrderNumber: "ORD-3",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  5000,
			PlacedAt:    &now,
		},

		// Previous Period (100.00, 1 order, 1 customer)
		{
			UserID:      helpers.Customer2UserID,
			OrderNumber: "ORD-4",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  10000,
			PlacedAt:    &fortyDaysAgo,
		},
	}

	for _, o := range orders {
		s.Require().NoError(s.container.DB.Create(&o).Error)
	}

	res := s.adminClient.Get(s.T(), "/api/report/summary")
	s.Require().Equal(http.StatusOK, res.Code)

	var responseBody struct {
		Data reportModel.ReportSummaryResponse `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &responseBody))

	// Assert Revenue (300 vs 100) -> up (+200%)
	s.Equal(300.0, responseBody.Data.TotalRevenue.Value)
	s.Equal(200.0, responseBody.Data.TotalRevenue.PercentageChange)
	s.Equal("up", responseBody.Data.TotalRevenue.Trend)

	// Assert Orders (3 vs 1) -> up (+200%)
	s.Equal(3, responseBody.Data.TotalOrders.Value)
	s.Equal(200.0, responseBody.Data.TotalOrders.PercentageChange)
	s.Equal("up", responseBody.Data.TotalOrders.Trend)

	// Assert Customers (2 vs 1) -> up (+100%)
	s.Equal(2, responseBody.Data.TotalCustomers.Value)
	s.Equal(100.0, responseBody.Data.TotalCustomers.PercentageChange)
	s.Equal("up", responseBody.Data.TotalCustomers.Trend)
}

func (s *ReportSuite) TestGetSummaryReport_WithDateFilters() {
	s.cleanupDomainData()
	now := time.Now()

	// Guarantee exact durations (30 * 24 * hour) instead of variable months
	oneMonthAgo := now.Add(-30 * 24 * time.Hour)
	twoMonthsAgo := now.Add(-60 * 24 * time.Hour)
	threeMonthsAgo := now.Add(-90 * 24 * time.Hour)

	orders := []orderEntity.Order{
		// "Current" Custom Period (Month -2 to Month -1) -> Total 200.00
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-1",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  20000,
			// Make sure it falls perfectly within [twoMonthsAgo, oneMonthAgo)
			PlacedAt: &twoMonthsAgo,
		},

		// "Previous" Custom Period (Month -3 to Month -2) -> Total 50.00
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-2",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  5000,
			// Make sure it falls perfectly within [threeMonthsAgo, twoMonthsAgo)
			PlacedAt: &threeMonthsAgo,
		},

		// Completely outside (Now) - shouldn't affect the filtered period
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-3",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  10000,
			PlacedAt:    &now,
		},
	}

	for _, o := range orders {
		s.Require().NoError(s.container.DB.Create(&o).Error)
	}

	// Request with custom dates
	startDate := twoMonthsAgo.Format(time.RFC3339)
	endDate := oneMonthAgo.Format(time.RFC3339)
	urlPath := "/api/report/summary?time_range=custom&start_date=" + url.QueryEscape(
		startDate,
	) + "&end_date=" + url.QueryEscape(
		endDate,
	)

	res := s.adminClient.Get(s.T(), urlPath)
	s.Require().Equal(http.StatusOK, res.Code)

	var responseBody struct {
		Data reportModel.ReportSummaryResponse `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &responseBody))

	// Assert Revenue in Custom "Current" window (200.00) vs "Prev" window (50.00) => +300%
	s.Equal(200.0, responseBody.Data.TotalRevenue.Value)
	s.Equal(300.0, responseBody.Data.TotalRevenue.PercentageChange)
	s.Equal("up", responseBody.Data.TotalRevenue.Trend)
}

func (s *ReportSuite) TestGetSummaryReport_LastXFilter() {
	s.cleanupDomainData()
	now := time.Now()

	fiveDaysAgo := now.AddDate(0, 0, -5)
	fifteenDaysAgo := now.AddDate(0, 0, -15)

	orders := []orderEntity.Order{
		// Inside Current Period (Last 10 days) -> Total = 200.00
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-1",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  20000,
			PlacedAt:    &fiveDaysAgo,
		},

		// Inside Previous Period (10 to 20 days ago) -> Total = 100.00
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-2",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  10000,
			PlacedAt:    &fifteenDaysAgo,
		},
	}

	for _, o := range orders {
		s.Require().NoError(s.container.DB.Create(&o).Error)
	}

	url := "/api/report/summary?time_range=last_x&last_x_amount=10&last_x_unit=day"
	res := s.adminClient.Get(s.T(), url)
	s.Require().Equal(http.StatusOK, res.Code)

	var responseBody struct {
		Data reportModel.ReportSummaryResponse `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &responseBody))

	// Assert Revenue "Last 10 Days" vs "10 days prior to that" => 200 vs 100 (+100%)
	s.Equal(200.0, responseBody.Data.TotalRevenue.Value)
	s.Equal(100.0, responseBody.Data.TotalRevenue.PercentageChange)
	s.Equal("vs previous 10 days", responseBody.Data.TotalRevenue.ComparisonText)

	// Additional filter variations requested by user
	urlHour := "/api/report/summary?time_range=last_x&last_x_amount=24&last_x_unit=hour"
	resHour := s.adminClient.Get(s.T(), urlHour)
	s.Require().Equal(http.StatusOK, resHour.Code)

	urlWeek := "/api/report/summary?time_range=last_x&last_x_amount=2&last_x_unit=week"
	resWeek := s.adminClient.Get(s.T(), urlWeek)
	s.Require().Equal(http.StatusOK, resWeek.Code)

	urlMonth := "/api/report/summary?time_range=last_x&last_x_amount=1&last_x_unit=month"
	resMonth := s.adminClient.Get(s.T(), urlMonth)
	s.Require().Equal(http.StatusOK, resMonth.Code)

	urlQuarter := "/api/report/summary?time_range=last_x&last_x_amount=1&last_x_unit=quarter"
	resQuarter := s.adminClient.Get(s.T(), urlQuarter)
	s.Require().Equal(http.StatusOK, resQuarter.Code)

	urlYear := "/api/report/summary?time_range=last_x&last_x_amount=1&last_x_unit=year"
	resYear := s.adminClient.Get(s.T(), urlYear)
	s.Require().Equal(http.StatusOK, resYear.Code)
}

func (s *ReportSuite) TestGetSummaryReport_Authorization() {
	s.cleanupDomainData()

	// 1. Unauthenticated (No Token)
	res := s.client.Get(s.T(), "/api/report/summary")
	s.Require().Equal(http.StatusUnauthorized, res.Code)

	// 2. Customer Token (Forbidden access, assuming Admin Auth middleware enforces Roles)
	res = s.customerClient.Get(s.T(), "/api/report/summary")
	s.Require().Equal(http.StatusForbidden, res.Code)

	// 3. Admin Token (Authorized)
	res = s.adminClient.Get(s.T(), "/api/report/summary")
	s.Require().Equal(http.StatusOK, res.Code)
}

func (s *ReportSuite) TestGetSummaryReport_EmptyState() {
	s.cleanupDomainData()

	// Querying a DB with zero orders
	res := s.adminClient.Get(s.T(), "/api/report/summary")
	s.Require().Equal(http.StatusOK, res.Code)

	var responseBody struct {
		Data reportModel.ReportSummaryResponse `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &responseBody))

	// Ensure no panics occurred and fields default to 0
	s.Equal(0.0, responseBody.Data.TotalRevenue.Value)
	s.Equal(0, responseBody.Data.TotalOrders.Value)
	s.Equal(0.0, responseBody.Data.AverageOrderValue.Value)
	s.Equal(0, responseBody.Data.TotalCustomers.Value)
}

func (s *ReportSuite) TestGetSummaryReport_CompareFalse() {
	s.cleanupDomainData()
	now := time.Now()

	orders := []orderEntity.Order{
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-1",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  10000,
			PlacedAt:    &now,
		},
	}
	for _, o := range orders {
		s.Require().NoError(s.container.DB.Create(&o).Error)
	}

	// Set compare=false manually
	res := s.adminClient.Get(s.T(), "/api/report/summary?compare=false")
	s.Require().Equal(http.StatusOK, res.Code)

	var responseBody struct {
		Data reportModel.ReportSummaryResponse `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &responseBody))

	// Without comparison, PercentageChange stays 0 and trend "flat"
	s.Equal(100.0, responseBody.Data.TotalRevenue.Value)
	s.Equal(0.0, responseBody.Data.TotalRevenue.PercentageChange)
	s.Equal("flat", responseBody.Data.TotalRevenue.Trend)
}

func (s *ReportSuite) TestGetSummaryReport_TimeRanges() {
	s.cleanupDomainData()
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	orders := []orderEntity.Order{
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-1",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  10000,
			PlacedAt:    &now,
		},
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-2",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  5000,
			PlacedAt:    &yesterday,
		},
	}
	for _, o := range orders {
		s.Require().NoError(s.container.DB.Create(&o).Error)
	}

	// Test "today"
	res := s.adminClient.Get(s.T(), "/api/report/summary?time_range=today")
	s.Require().Equal(http.StatusOK, res.Code)

	var bodyToday struct {
		Data reportModel.ReportSummaryResponse `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &bodyToday))

	// Today's total is 100.00
	s.Equal(100.0, bodyToday.Data.TotalRevenue.Value)

	// Test "yesterday"
	resYest := s.adminClient.Get(s.T(), "/api/report/summary?time_range=yesterday")
	s.Require().Equal(http.StatusOK, resYest.Code)

	var bodyYest struct {
		Data reportModel.ReportSummaryResponse `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(resYest.Body.Bytes(), &bodyYest))

	// Yesterday's total is 50.00
	s.Equal(50.0, bodyYest.Data.TotalRevenue.Value)

	// Test "this_week"
	resThisWeek := s.adminClient.Get(s.T(), "/api/report/summary?time_range=this_week")
	s.Require().Equal(http.StatusOK, resThisWeek.Code)

	// Test "this_month"
	resThisMonth := s.adminClient.Get(s.T(), "/api/report/summary?time_range=this_month")
	s.Require().Equal(http.StatusOK, resThisMonth.Code)

	// Test "this_quarter"
	resThisQuarter := s.adminClient.Get(s.T(), "/api/report/summary?time_range=this_quarter")
	s.Require().Equal(http.StatusOK, resThisQuarter.Code)

	// Test "this_year"
	resThisYear := s.adminClient.Get(s.T(), "/api/report/summary?time_range=this_year")
	s.Require().Equal(http.StatusOK, resThisYear.Code)
}

func (s *ReportSuite) TestGetSummaryReport_InvalidFilters() {
	s.cleanupDomainData()

	// Invalid Start Date custom param
	res := s.adminClient.Get(s.T(), "/api/report/summary?time_range=custom&start_date=2024-invalid")
	s.Require().Equal(http.StatusBadRequest, res.Code)

	// last_x with negative amount
	res2 := s.adminClient.Get(
		s.T(),
		"/api/report/summary?time_range=last_x&last_x_amount=-5&last_x_unit=day",
	)
	s.Require().Equal(http.StatusBadRequest, res2.Code)
}
