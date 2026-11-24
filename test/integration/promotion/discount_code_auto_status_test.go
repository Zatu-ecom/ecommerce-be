package promotion_test

import (
	"net/http"
	"time"

	promotionEntity "ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/factory/singleton"
	"ecommerce-be/test/integration/helpers"
)

func (s *DiscountCodeTestSuite) TestCreateFutureDiscountCodeIsInactive() {
	payload := helpers.NewDiscountCodePayload("FUTURE1").
		Title("Future Code").
		Percentage(10).
		StartsAt(helpers.FutureRFC3339(2)).
		EndsAt(helpers.FutureRFC3339(30)).
		Build()

	res := s.sellerClient.Post(s.T(), DiscountCodeAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)

	s.Equal(false, dc["isActive"])
	s.Equal(true, dc["autoStart"])
	s.Equal(true, dc["autoEnd"])
}

func (s *DiscountCodeTestSuite) TestAutoStartDiscountCodeViaCronSweep() {
	payload := helpers.NewDiscountCodePayload("AUTOSTART1").
		Title("Auto Start").
		Percentage(12).
		StartsAt(helpers.FutureRFC3339(2)).
		EndsAt(helpers.FutureRFC3339(30)).
		Build()

	res := s.sellerClient.Post(s.T(), DiscountCodeAPIEndpoint, payload)
	s.Require().Equal(http.StatusCreated, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	id := uint(dc["id"].(float64))
	s.Equal(false, dc["isActive"])

	// Simulate wall-clock reaching starts_at.
	pastStart := time.Now().UTC().Add(-time.Hour)
	s.Require().NoError(
		s.container.DB.Model(&promotionEntity.DiscountCode{}).
			Where("id = ?", id).
			Update("starts_at", pastStart).Error,
	)

	singleton.GetInstance().GetPromotionCronService().SweepStatusTransitions()

	res = s.sellerClient.Get(s.T(), discountCodeURL(id))
	s.Require().Equal(http.StatusOK, res.Code)
	response = helpers.ParseResponse(s.T(), res.Body)
	dc = response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal(true, dc["isActive"])
}

func (s *DiscountCodeTestSuite) TestAutoEndDiscountCodeViaCronSweep() {
	id := s.createDiscountCode(s.sellerClient, "AUTOEND1")

	pastEnd := time.Now().UTC().Add(-time.Minute)
	s.Require().NoError(
		s.container.DB.Model(&promotionEntity.DiscountCode{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"ends_at":  pastEnd,
				"auto_end": true,
			}).Error,
	)

	singleton.GetInstance().GetPromotionCronService().SweepStatusTransitions()

	res := s.sellerClient.Get(s.T(), discountCodeURL(id))
	s.Require().Equal(http.StatusOK, res.Code)
	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal(false, dc["isActive"])
}

func (s *DiscountCodeTestSuite) TestManualDeactivateNotReactivatedByCron() {
	id := s.createDiscountCode(s.sellerClient, "MANUALOFF")

	res := s.sellerClient.Patch(
		s.T(),
		discountCodeStatusURL(id),
		map[string]any{"isActive": false},
	)
	s.Require().Equal(http.StatusOK, res.Code)

	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal(false, dc["isActive"])
	s.Equal(false, dc["autoStart"])

	singleton.GetInstance().GetPromotionCronService().SweepStatusTransitions()

	res = s.sellerClient.Get(s.T(), discountCodeURL(id))
	s.Require().Equal(http.StatusOK, res.Code)
	response = helpers.ParseResponse(s.T(), res.Body)
	dc = response["data"].(map[string]any)["discountCode"].(map[string]any)
	s.Equal(false, dc["isActive"])
	s.Equal(false, dc["autoStart"])
}
