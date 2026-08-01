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
	loc := now.Location()

	// Use two calendar days guaranteed to fall inside this_month's label range.
	// On the 1st, "yesterday" is in the prior month and is excluded from
	// this_month — so anchor to the 1st and 2nd instead of now-1/now.
	earlierDay := time.Date(now.Year(), now.Month(), 1, 10, 0, 0, 0, loc)
	laterDay := earlierDay.AddDate(0, 0, 1)
	if now.Day() > 1 {
		laterDay = time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, loc)
		earlierDay = laterDay.AddDate(0, 0, -1)
	}

	orders := []orderEntity.Order{
		helpers.NewOrderEntity().
			UserID(helpers.CustomerUserID).
			OrderNumber("ORD-TREND-1").
			Completed().
			TotalCents(10000).
			PlacedAt(laterDay).
			Build(),
		helpers.NewOrderEntity().
			UserID(helpers.CustomerUserID).
			OrderNumber("ORD-TREND-2").
			Confirmed().
			TotalCents(5000).
			PlacedAt(laterDay).
			Build(),
		helpers.NewOrderEntity().
			UserID(helpers.Seller2UserID).
			OrderNumber("ORD-TREND-3").
			Completed().
			TotalCents(5000).
			PlacedAt(earlierDay).
			Build(),
		helpers.NewOrderEntity().
			UserID(helpers.Customer2UserID).
			OrderNumber("ORD-TREND-4").
			Cancelled().
			TotalCents(10000).
			PlacedAt(laterDay.AddDate(0, 0, -2)).
			Build(),
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

	earlierDayStr := earlierDay.Format("2006-01-02")
	laterDayStr := laterDay.Format("2006-01-02")

	earlierDayFound := false
	laterDayFound := false

	for i, label := range responseBody.Data.Labels {
		if label == laterDayStr {
			laterDayFound = true
			s.Equal(150.0, responseBody.Data.RevenueData[i])
			s.Equal(2, responseBody.Data.OrderVolumeData[i])
		} else if label == earlierDayStr {
			earlierDayFound = true
			s.Equal(50.0, responseBody.Data.RevenueData[i])
			s.Equal(1, responseBody.Data.OrderVolumeData[i])
		}
	}

	s.True(laterDayFound, "Later day's data should be in the trends")
	s.True(earlierDayFound, "Earlier day's data should be in the trends")
}

func (s *ReportSuite) TestGetSalesTrends_Today_Hourly() {
	s.cleanupDomainData()
	now := time.Now()
	twoHoursAgo := now.Add(-2 * time.Hour)
	twentyHoursAgo := now.Add(-20 * time.Hour)

	orders := []orderEntity.Order{
		helpers.NewOrderEntity().
			UserID(helpers.CustomerUserID).
			OrderNumber("ORD-HR-1").
			Completed().
			TotalCents(20000).
			PlacedAt(now).
			Build(),
		helpers.NewOrderEntity().
			UserID(helpers.CustomerUserID).
			OrderNumber("ORD-HR-2").
			Completed().
			TotalCents(10000).
			PlacedAt(twoHoursAgo).
			Build(),
		helpers.NewOrderEntity().
			UserID(helpers.CustomerUserID).
			OrderNumber("ORD-HR-3").
			Completed().
			TotalCents(5000).
			PlacedAt(twentyHoursAgo).
			Build(),
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

	// `time_range=today` includes [00:00 local, now]. An event placed 2h/20h
	// before `now` only belongs to today when `now.Hour() >= 2` or >= 20 in
	// the caller's local timezone. Otherwise the order's placed_at is on
	// yesterday and must be filtered out — which is correct backend behaviour.
	twoHoursAgoInToday := twoHoursAgo.Day() == now.Day()
	twentyHoursAgoInToday := twentyHoursAgo.Day() == now.Day()

	nowFound := false
	for i, label := range responseBody.Data.Labels {
		switch label {
		case nowStr:
			s.Equal(200.0, responseBody.Data.RevenueData[i])
			s.Equal(1, responseBody.Data.OrderVolumeData[i])
			nowFound = true
		case twoHoursAgoStr:
			if twoHoursAgoInToday {
				s.Equal(100.0, responseBody.Data.RevenueData[i])
				s.Equal(1, responseBody.Data.OrderVolumeData[i])
			} else {
				s.Equal(0.0, responseBody.Data.RevenueData[i],
					"2h-ago event is from yesterday; should not count for today")
				s.Equal(0, responseBody.Data.OrderVolumeData[i])
			}
		case twentyHoursAgoStr:
			if twentyHoursAgoInToday {
				s.Equal(50.0, responseBody.Data.RevenueData[i])
				s.Equal(1, responseBody.Data.OrderVolumeData[i])
			} else {
				s.Equal(0.0, responseBody.Data.RevenueData[i],
					"20h-ago event is from yesterday; should not count for today")
				s.Equal(0, responseBody.Data.OrderVolumeData[i])
			}
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
