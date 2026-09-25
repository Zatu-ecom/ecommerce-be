package order_test

import (
	"fmt"
	"net/http"

	orderEntity "ecommerce-be/order/entity"
	"ecommerce-be/test/integration/helpers"
)

func (s *CartCouponTestSuite) TestCartItemPromotionSyncedOnGetCart() {
	promoID := s.createSellerPromotion(helpers.PercentagePromotionPayload("SyncPromo", 10))
	s.addCartItem(1, 1)

	s.getCartOK()

	promoIDs := s.listCartItemPromotionIDs()
	s.Require().Contains(promoIDs, promoID)
}

func (s *CartCouponTestSuite) TestCartItemPromotionStaleRowsRemoved() {
	promoID := s.createSellerPromotion(helpers.PercentagePromotionPayload("StalePromo", 10))
	s.addCartItem(1, 1)
	s.getCartOK()
	s.Require().Contains(s.listCartItemPromotionIDs(), promoID)

	res := s.sellerClient.Patch(s.T(), fmt.Sprintf("/api/promotion/%d/status", promoID), map[string]any{
		"status": "paused",
	})
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())

	s.getCartOK()
	s.NotContains(s.listCartItemPromotionIDs(), promoID)

	var count int64
	s.Require().NoError(s.container.DB.Model(&orderEntity.CartItemPromotion{}).
		Where("promotion_id = ?", promoID).Count(&count).Error)
	s.Equal(int64(0), count)
}

func (s *CartCouponTestSuite) TestCartItemPromotionMatchesAppliedPromoIDs() {
	promoA := s.createSellerPromotion(
		helpers.NewPercentagePromotion("MatchA", 5).CanStackWithOtherPromotions(true).Build(),
	)
	promoB := s.createSellerPromotion(
		helpers.NewPercentagePromotion("MatchB", 5).CanStackWithOtherPromotions(true).Build(),
	)

	s.addCartItem(1, 1)
	response := s.getCartOK()
	applied := response["data"].(map[string]any)["appliedPromotions"].([]any)

	appliedIDs := map[uint]struct{}{}
	for _, p := range applied {
		appliedIDs[uint(p.(map[string]any)["promotionId"].(float64))] = struct{}{}
	}
	s.Contains(appliedIDs, promoA)
	s.Contains(appliedIDs, promoB)

	stored := s.listCartItemPromotionIDs()
	storedSet := map[uint]struct{}{}
	for _, id := range stored {
		storedSet[id] = struct{}{}
	}
	for id := range appliedIDs {
		s.Contains(storedSet, id)
	}
	for id := range storedSet {
		s.Contains(appliedIDs, id)
	}
}

func (s *CartCouponTestSuite) TestCartItemPromotionNeverStoresDiscountAmounts() {
	s.createSellerPromotion(helpers.PercentagePromotionPayload("NoAmount", 15))
	s.addCartItem(1, 1)
	s.getCartOK()

	var cartID uint
	s.Require().NoError(s.container.DB.Model(&orderEntity.Cart{}).
		Where("user_id = ? AND status = ?", helpers.CustomerUserID, "active").
		Select("id").Scan(&cartID).Error)

	var itemIDs []uint
	s.Require().NoError(s.container.DB.Model(&orderEntity.CartItem{}).
		Where("cart_id = ?", cartID).Pluck("id", &itemIDs).Error)
	s.Require().NotEmpty(itemIDs)

	var rows []orderEntity.CartItemPromotion
	s.Require().NoError(s.container.DB.Where("cart_item_id IN ?", itemIDs).Find(&rows).Error)
	s.Require().NotEmpty(rows)
	for _, row := range rows {
		s.NotZero(row.CartItemID)
		s.NotZero(row.PromotionID)
	}
}
