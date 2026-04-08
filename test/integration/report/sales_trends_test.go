package report_test

import (
	"encoding/json"
	"net/http"
	"time"

	orderEntity "ecommerce-be/order/entity"
	reportModel "ecommerce-be/report/model"
	"ecommerce-be/test/integration/helpers"
)

func (s *ReportSuite) TestGetSalesTrends_ThisMonth() {
	s.cleanupDomainData()
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	twoDaysAgo := now.AddDate(0, 0, -2)

	orders := []orderEntity.Order{
		{
			// Today: 2 orders, 150 total
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-TREND-1",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  10000,
			PlacedAt:    &now,
		},
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-TREND-2",
			Status:      orderEntity.ORDER_STATUS_CONFIRMED,
			TotalCents:  5000,
			PlacedAt:    &now,
		},
		{
			// Yesterday: 1 order, 50 total
			UserID:      helpers.Seller2UserID,
			OrderNumber: "ORD-TREND-3",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  5000,
			PlacedAt:    &yesterday,
		},
		{
			// Two days ago: 1 order, 100 total (Cancelled -> should ignore)
			UserID:      helpers.Customer2UserID,
			OrderNumber: "ORD-TREND-4",
			Status:      orderEntity.ORDER_STATUS_CANCELLED,
			TotalCents:  10000,
			PlacedAt:    &twoDaysAgo,
		},
	}

	for _, o := range orders {
		s.Require().NoError(s.container.DB.Create(&o).Error)
	}

	res := s.adminClient.Get(s.T(), "/api/report/sales/trends?time_range=this_month")
	s.Require().Equal(http.StatusOK, res.Code)

	var responseBody struct {
		Data reportModel.ReportTrendsResponse `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &responseBody))

	// Assert Response Structure
	s.Equal("day", responseBody.Data.Interval)
	s.NotEmpty(responseBody.Data.Labels)
	s.Equal(len(responseBody.Data.Labels), len(responseBody.Data.RevenueData))
	s.Equal(len(responseBody.Data.Labels), len(responseBody.Data.OrderVolumeData))

	// Track the specific dates formatted as YYYY-MM-DD
	todayStr := now.Format("2006-01-02")
	yesterdayStr := yesterday.Format("2006-01-02")

	todayFound := false
	yesterdayFound := false

	for i, label := range responseBody.Data.Labels {
		if label == todayStr {
			todayFound = true
			s.Equal(150.0, responseBody.Data.RevenueData[i])
			s.Equal(2, responseBody.Data.OrderVolumeData[i])
		} else if label == yesterdayStr {
			yesterdayFound = true
			s.Equal(50.0, responseBody.Data.RevenueData[i])
			s.Equal(1, responseBody.Data.OrderVolumeData[i])
		}
	}

	s.True(todayFound, "Today's data should be in the trends")
	s.True(yesterdayFound, "Yesterday's data should be in the trends")
}

func (s *ReportSuite) TestGetSalesTrends_Today_Hourly() {
	s.cleanupDomainData()
	now := time.Now()
	twoHoursAgo := now.Add(-2 * time.Hour)
	twentyHoursAgo := now.Add(-20 * time.Hour)

	orders := []orderEntity.Order{
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-HR-1",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  20000,
			PlacedAt:    &now,
		},
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-HR-2",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  10000,
			PlacedAt:    &twoHoursAgo,
		},
		{
			UserID:      helpers.CustomerUserID,
			OrderNumber: "ORD-HR-3",
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  5000,
			PlacedAt:    &twentyHoursAgo,
		},
	}

	for _, o := range orders {
		s.Require().NoError(s.container.DB.Create(&o).Error)
	}

	res := s.adminClient.Get(s.T(), "/api/report/sales/trends?time_range=today")
	s.Require().Equal(http.StatusOK, res.Code)

	var responseBody struct {
		Data reportModel.ReportTrendsResponse `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &responseBody))

	// Assert Hourly Interval
	s.Equal("hour", responseBody.Data.Interval)
	s.Len(responseBody.Data.Labels, 24)

	// Since we use 24 hours, let's verify lengths match
	s.Equal(24, len(responseBody.Data.RevenueData))
	s.Equal(24, len(responseBody.Data.OrderVolumeData))

	nowStr := now.Format("15:00")
	twoHoursAgoStr := twoHoursAgo.Format("15:00")
	twentyHoursAgoStr := twentyHoursAgo.Format("15:00")

	nowFound := false
	for i, label := range responseBody.Data.Labels {
		if label == nowStr {
			s.Equal(200.0, responseBody.Data.RevenueData[i])
			s.Equal(1, responseBody.Data.OrderVolumeData[i])
			nowFound = true
		} else if label == twoHoursAgoStr {
			s.Equal(100.0, responseBody.Data.RevenueData[i])
			s.Equal(1, responseBody.Data.OrderVolumeData[i])
		} else if label == twentyHoursAgoStr {
			s.Equal(50.0, responseBody.Data.RevenueData[i])
			s.Equal(1, responseBody.Data.OrderVolumeData[i])
		}
	}
	s.True(nowFound, "Current hour should be represented")
}

func (s *ReportSuite) TestGetSalesTrends_EmptyState() {
	s.cleanupDomainData()

	res := s.adminClient.Get(s.T(), "/api/report/sales/trends?time_range=this_year")
	s.Require().Equal(http.StatusOK, res.Code)

	var responseBody struct {
		Data reportModel.ReportTrendsResponse `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &responseBody))

	// Year should resolve to "month" interval
	s.Equal("month", responseBody.Data.Interval)
	s.Equal(12, len(responseBody.Data.Labels))
	s.Equal(12, len(responseBody.Data.RevenueData))
	s.Equal(12, len(responseBody.Data.OrderVolumeData))

	// Assert everything is zero since no orders were placed
	for i := range responseBody.Data.Labels {
		s.Equal(0.0, responseBody.Data.RevenueData[i])
		s.Equal(0, responseBody.Data.OrderVolumeData[i])
	}
}

func (s *ReportSuite) TestGetSalesTrends_Authorization() {
	s.cleanupDomainData()

	res := s.client.Get(s.T(), "/api/report/sales/trends")
	s.Require().Equal(http.StatusUnauthorized, res.Code)

	resForbid := s.customerClient.Get(s.T(), "/api/report/sales/trends")
	s.Require().Equal(http.StatusForbidden, resForbid.Code)
}
