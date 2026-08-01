package order_test

import (
	"fmt"
	"net/http"
	"testing"

	orderEntity "ecommerce-be/order/entity"
	promotionEntity "ecommerce-be/promotion/entity"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/suite"
)

type CartCouponTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	sellerClient    *helpers.APIClient
	customerClient  *helpers.APIClient
	anonymousClient *helpers.APIClient
}

func (s *CartCouponTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	s.sellerClient = helpers.NewAPIClient(s.server)
	s.sellerClient.SetToken(helpers.Login(
		s.T(), s.sellerClient, helpers.Seller2Email, helpers.Seller2Password,
	))

	s.customerClient = helpers.NewAPIClient(s.server)
	s.customerClient.SetToken(helpers.Login(
		s.T(), s.customerClient, helpers.CustomerEmail, helpers.CustomerPassword,
	))

	s.anonymousClient = helpers.NewAPIClient(s.server)
}

func (s *CartCouponTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *CartCouponTestSuite) SetupTest() {
	s.cleanupCartCouponsAndCodes()
	s.cleanupSellerPromotions()
	s.cleanupCarts()
}

func TestCartCouponAPI(t *testing.T) {
	suite.Run(t, new(CartCouponTestSuite))
}

func (s *CartCouponTestSuite) cleanupCarts() {
	userIDs := []uint{helpers.CustomerUserID}
	var cartIDs []uint
	err := s.container.DB.Model(&orderEntity.Cart{}).
		Where("user_id IN ?", userIDs).
		Pluck("id", &cartIDs).Error
	s.Require().NoError(err)
	if len(cartIDs) == 0 {
		return
	}
	var itemIDs []uint
	s.Require().NoError(s.container.DB.Model(&orderEntity.CartItem{}).
		Where("cart_id IN ?", cartIDs).
		Pluck("id", &itemIDs).Error)
	if len(itemIDs) > 0 {
		s.Require().NoError(s.container.DB.Where("cart_item_id IN ?", itemIDs).
			Delete(&orderEntity.CartItemPromotion{}).Error)
	}
	s.Require().NoError(s.container.DB.Where("cart_id IN ?", cartIDs).Delete(&orderEntity.CartAppliedCoupon{}).Error)
	s.Require().NoError(s.container.DB.Where("cart_id IN ?", cartIDs).Delete(&orderEntity.CartItem{}).Error)
	s.Require().NoError(s.container.DB.Where("id IN ?", cartIDs).Delete(&orderEntity.Cart{}).Error)
}

func (s *CartCouponTestSuite) cleanupSellerPromotions() {
	var promoIDs []uint
	err := s.container.DB.Model(&promotionEntity.Promotion{}).
		Where("seller_id = ?", helpers.Seller2UserID).
		Pluck("id", &promoIDs).Error
	s.Require().NoError(err)
	if len(promoIDs) == 0 {
		return
	}
	s.Require().NoError(s.container.DB.Where("promotion_id IN ?", promoIDs).
		Delete(&promotionEntity.PromotionUsage{}).Error)
	s.Require().NoError(s.container.DB.Where("promotion_id IN ?", promoIDs).
		Delete(&promotionEntity.PromotionProductVariant{}).Error)
	s.Require().NoError(s.container.DB.Where("promotion_id IN ?", promoIDs).
		Delete(&promotionEntity.PromotionProduct{}).Error)
	s.Require().NoError(s.container.DB.Where("promotion_id IN ?", promoIDs).
		Delete(&promotionEntity.PromotionCategory{}).Error)
	s.Require().NoError(s.container.DB.Where("promotion_id IN ?", promoIDs).
		Delete(&promotionEntity.PromotionCollection{}).Error)
	s.Require().NoError(s.container.DB.Unscoped().Where("id IN ?", promoIDs).
		Delete(&promotionEntity.Promotion{}).Error)
}

func (s *CartCouponTestSuite) cleanupCartCouponsAndCodes() {
	sellerIDs := []uint{helpers.Seller2UserID}
	var codeIDs []uint
	err := s.container.DB.Model(&promotionEntity.DiscountCode{}).
		Where("seller_id IN ?", sellerIDs).
		Pluck("id", &codeIDs).Error
	s.Require().NoError(err)
	if len(codeIDs) == 0 {
		return
	}
	s.Require().NoError(s.container.DB.Where("discount_code_id IN ?", codeIDs).Delete(&orderEntity.CartAppliedCoupon{}).Error)
	s.Require().NoError(s.container.DB.Where("discount_code_id IN ?", codeIDs).Delete(&promotionEntity.DiscountCodeUsage{}).Error)
	s.Require().NoError(s.container.DB.Where("discount_code_id IN ?", codeIDs).Delete(&promotionEntity.DiscountCodeProduct{}).Error)
	s.Require().NoError(s.container.DB.Where("discount_code_id IN ?", codeIDs).Delete(&promotionEntity.DiscountCodeCategory{}).Error)
	s.Require().NoError(s.container.DB.Where("discount_code_id IN ?", codeIDs).Delete(&promotionEntity.DiscountCodeCollection{}).Error)
	s.Require().NoError(s.container.DB.Where("id IN ?", codeIDs).Delete(&promotionEntity.DiscountCode{}).Error)
}

func (s *CartCouponTestSuite) addCartItem(variantID uint, qty int) {
	res := s.customerClient.Post(s.T(), "/api/order/cart/item", helpers.AddCartItemsPayload(variantID, qty))
	s.Require().Equal(http.StatusCreated, res.Code, res.Body.String())
}

func (s *CartCouponTestSuite) createSellerDiscountCode(payload map[string]any) uint {
	res := s.sellerClient.Post(s.T(), "/api/promotion/discount-code", payload)
	s.Require().Equal(http.StatusCreated, res.Code, res.Body.String())
	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	return uint(dc["id"].(float64))
}

func (s *CartCouponTestSuite) percentageCouponPayload(code string, value int64) map[string]any {
	return helpers.PercentageDiscountCode(code, value)
}

func (s *CartCouponTestSuite) applyCouponOK(code string) map[string]any {
	res := s.customerClient.Post(s.T(), CartCouponAPIEndpoint, helpers.ApplyCouponPayload(code))
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())
	return helpers.ParseResponse(s.T(), res.Body)
}

func (s *CartCouponTestSuite) getCartOK() map[string]any {
	res := s.customerClient.Get(s.T(), "/api/order/cart")
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())
	return helpers.ParseResponse(s.T(), res.Body)
}

func (s *CartCouponTestSuite) deactivateDiscountCode(id uint) {
	res := s.sellerClient.Patch(s.T(), fmt.Sprintf("/api/promotion/discount-code/%d/status", id), map[string]any{
		"isActive": false,
	})
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())
}

func (s *CartCouponTestSuite) updateDiscountCode(id uint, payload map[string]any) {
	res := s.sellerClient.Put(s.T(), fmt.Sprintf("/api/promotion/discount-code/%d", id), payload)
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())
}

func (s *CartCouponTestSuite) createSellerPromotion(payload map[string]any) uint {
	res := s.sellerClient.Post(s.T(), "/api/promotion", payload)
	s.Require().Equal(http.StatusCreated, res.Code, res.Body.String())
	response := helpers.ParseResponse(s.T(), res.Body)
	promo := response["data"].(map[string]any)["promotion"].(map[string]any)
	return uint(promo["id"].(float64))
}

func (s *CartCouponTestSuite) percentagePromotionPayload(name string, percentage float64) map[string]any {
	return helpers.PercentagePromotionPayload(name, percentage)
}

func (s *CartCouponTestSuite) addDiscountCodeProducts(codeID uint, productIDs ...uint) {
	res := s.sellerClient.Post(s.T(), "/api/promotion/discount-code/scope/product", map[string]any{
		"discountCodeId": codeID,
		"productIds":     productIDs,
	})
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())
}

func (s *CartCouponTestSuite) countCartAppliedCoupons() int64 {
	var cartID uint
	err := s.container.DB.Model(&orderEntity.Cart{}).
		Where("user_id = ? AND status = ?", helpers.CustomerUserID, "active").
		Select("id").Scan(&cartID).Error
	s.Require().NoError(err)
	if cartID == 0 {
		return 0
	}
	var count int64
	s.Require().NoError(s.container.DB.Model(&orderEntity.CartAppliedCoupon{}).
		Where("cart_id = ?", cartID).Count(&count).Error)
	return count
}

func (s *CartCouponTestSuite) listCartItemPromotionIDs() []uint {
	var cartID uint
	err := s.container.DB.Model(&orderEntity.Cart{}).
		Where("user_id = ? AND status = ?", helpers.CustomerUserID, "active").
		Select("id").Scan(&cartID).Error
	s.Require().NoError(err)
	if cartID == 0 {
		return nil
	}
	var itemIDs []uint
	s.Require().NoError(s.container.DB.Model(&orderEntity.CartItem{}).
		Where("cart_id = ?", cartID).Pluck("id", &itemIDs).Error)
	if len(itemIDs) == 0 {
		return nil
	}
	var promoIDs []uint
	s.Require().NoError(s.container.DB.Model(&orderEntity.CartItemPromotion{}).
		Where("cart_item_id IN ?", itemIDs).Pluck("promotion_id", &promoIDs).Error)
	return promoIDs
}

func couponURL(code string) string {
	return fmt.Sprintf("%s/%s", CartCouponAPIEndpoint, code)
}
