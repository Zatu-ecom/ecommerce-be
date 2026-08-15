package report_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	orderEntity "ecommerce-be/order/entity"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/suite"
)

// ReportMoneySuite verifies the one-money-language contract on report
// responses (US4): revenue/discount metrics are Money objects, not float
// /100 twins, and carry amount/amountCents/formatted.
type ReportMoneySuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler
	admin     *helpers.APIClient
}

func (s *ReportMoneySuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)
	s.admin = helpers.NewAPIClient(s.server)
	s.admin.SetToken(helpers.Login(s.T(), s.admin, helpers.AdminEmail, helpers.AdminPassword))
}

func (s *ReportMoneySuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func TestReportMoneyMetrics(t *testing.T) {
	suite.Run(t, new(ReportMoneySuite))
}

func (s *ReportMoneySuite) cleanupOrders() {
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_item`).Error)
	s.Require().NoError(s.container.DB.Where("1=1").Delete(&orderEntity.Order{}).Error)
}

// ─── Summary revenue uses the Money shape ───────────────────────────────────
func (s *ReportMoneySuite) TestSummaryRevenueIsMoney() {
	s.cleanupOrders()
	now := time.Now()
	s.Require().NoError(s.container.DB.Create(&orderEntity.Order{
		UserID:      helpers.Seller2UserID,
		OrderNumber: "MONEY-ORD-1",
		Status:      orderEntity.ORDER_STATUS_COMPLETED,
		TotalCents:  129950,
		PlacedAt:    &now,
	}).Error)

	res := s.admin.Get(s.T(), "/api/report/summary")
	s.Require().Equal(http.StatusOK, res.Code)

	var responseBody struct {
		Data map[string]any `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &responseBody))

	revenue := responseBody.Data["total_revenue"].(map[string]any)
	helpers.AssertMoney(s.T(), revenue["value"])
	helpers.AssertMoneyCents(s.T(), revenue["value"], 129950)
	helpers.AssertMoneyAmount(s.T(), revenue["value"], 1299.50)
}

// ─── AverageOrderValue uses the Money shape ─────────────────────────────────
func (s *ReportMoneySuite) TestAOVIsMoney() {
	s.cleanupOrders()
	now := time.Now()
	for i := 0; i < 2; i++ {
		s.Require().NoError(s.container.DB.Create(&orderEntity.Order{
			UserID:      helpers.Seller2UserID,
			OrderNumber: "AOV-ORD-" + string(rune('1'+i)),
			Status:      orderEntity.ORDER_STATUS_COMPLETED,
			TotalCents:  10000,
			PlacedAt:    &now,
		}).Error)
	}

	res := s.admin.Get(s.T(), "/api/report/summary")
	s.Require().Equal(http.StatusOK, res.Code)

	var responseBody struct {
		Data map[string]any `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &responseBody))

	aov := responseBody.Data["average_order_value"].(map[string]any)
	helpers.AssertMoney(s.T(), aov["value"])
	helpers.AssertMoneyCents(s.T(), aov["value"], 10000)
}

// ─── Trends revenue entries use Money shape ─────────────────────────────────
func (s *ReportMoneySuite) TestTrendsRevenueIsMoney() {
	s.cleanupOrders()
	now := time.Now()
	s.Require().NoError(s.container.DB.Create(&orderEntity.Order{
		UserID:      helpers.Seller2UserID,
		OrderNumber: "TREND-ORD-1",
		Status:      orderEntity.ORDER_STATUS_COMPLETED,
		TotalCents:  7500,
		PlacedAt:    &now,
	}).Error)

	res := s.admin.Get(s.T(), "/api/report/sales/trends?time_range=today")
	s.Require().Equal(http.StatusOK, res.Code)

	var responseBody struct {
		Data map[string]any `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &responseBody))

	revenueData, ok := responseBody.Data["revenue_data"].([]any)
	s.Require().True(ok, "revenue_data should be an array")
	s.NotEmpty(revenueData)

	for i, entry := range revenueData {
		helpers.AssertMoney(s.T(), entry)
		rev, _ := entry.(map[string]any)
		_, hasAmountCents := rev["amountCents"]
		s.True(hasAmountCents, "revenue_data[%d] must have amountCents", i)
	}
}

// ─── No bare *Cents scalars leak in responses ───────────────────────────────
func (s *ReportMoneySuite) TestNoBareCentsInResponse() {
	s.cleanupOrders()
	now := time.Now()
	s.Require().NoError(s.container.DB.Create(&orderEntity.Order{
		UserID:      helpers.Seller2UserID,
		OrderNumber: "NOBARE-ORD-1",
		Status:      orderEntity.ORDER_STATUS_COMPLETED,
		TotalCents:  15000,
		PlacedAt:    &now,
	}).Error)

	summaryRes := s.admin.Get(s.T(), "/api/report/summary")
	s.Require().Equal(http.StatusOK, summaryRes.Code)
	body := summaryRes.Body.String()
	s.NotContains(body, `"total_revenue_cents"`, "no bare total_revenue_cents key")
	s.NotContains(body, `"revenueCents"`, "no bare revenueCents key")

	trendsRes := s.admin.Get(s.T(), "/api/report/sales/trends?time_range=today")
	s.Require().Equal(http.StatusOK, trendsRes.Code)
	trendsBody := trendsRes.Body.String()
	s.NotContains(trendsBody, `"revenueCents"`, "no bare revenueCents key in trends")
}

// ─── Non-monetary fields stay plain, not Money objects ──────────────────────
func (s *ReportMoneySuite) TestNonMonetaryFieldsArePlain() {
	s.cleanupOrders()
	now := time.Now()
	s.Require().NoError(s.container.DB.Create(&orderEntity.Order{
		UserID:      helpers.Seller2UserID,
		OrderNumber: "PLAIN-ORD-1",
		Status:      orderEntity.ORDER_STATUS_COMPLETED,
		TotalCents:  5000,
		PlacedAt:    &now,
	}).Error)

	// Summary
	res := s.admin.Get(s.T(), "/api/report/summary")
	s.Require().Equal(http.StatusOK, res.Code)
	var summaryBody struct {
		Data map[string]any `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &summaryBody))

	totalOrders := summaryBody.Data["total_orders"].(map[string]any)
	_, isOrdersMoneyObj := totalOrders["value"].(map[string]any)
	s.False(isOrdersMoneyObj, "total_orders value should be plain int, not Money object")

	// Trends — order_volume_data items are plain numbers
	trendsRes := s.admin.Get(s.T(), "/api/report/sales/trends?time_range=today")
	s.Require().Equal(http.StatusOK, trendsRes.Code)
	var trendsBody struct {
		Data map[string]any `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(trendsRes.Body.Bytes(), &trendsBody))

	orderVolume, ok := trendsBody.Data["order_volume_data"].([]any)
	s.Require().True(ok, "order_volume_data should be an array")
	for i, v := range orderVolume {
		_, isNum := v.(float64)
		s.True(isNum, "order_volume_data[%d] should be a plain number, not an object", i)
	}
}
