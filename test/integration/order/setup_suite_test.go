package order_test

import (
	"net/http"
	"testing"

	orderEntity "ecommerce-be/order/entity"
	promotionEntity "ecommerce-be/promotion/entity"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/suite"
)

const (
	OrderAPIEndpoint       = "/api/order"
	OrderByIDAPIEndpoint   = "/api/order/%d"
	OrderStatusAPIEndpoint = "/api/order/%d/status"
	OrderCancelAPIEndpoint = "/api/order/%d/cancel"
)

// OrderSuite holds all shared state for order integration tests.
type OrderSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	client         *helpers.APIClient
	customerClient *helpers.APIClient
	sellerClient   *helpers.APIClient
	adminClient    *helpers.APIClient
}

func (s *OrderSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	s.client = helpers.NewAPIClient(s.server)

	s.customerClient = helpers.NewAPIClient(s.server)
	customerToken := helpers.Login(
		s.T(),
		s.customerClient,
		helpers.CustomerEmail,
		helpers.CustomerPassword,
	)
	s.customerClient.SetToken(customerToken)

	s.sellerClient = helpers.NewAPIClient(s.server)
	sellerToken := helpers.Login(
		s.T(),
		s.sellerClient,
		helpers.Seller2Email,
		helpers.Seller2Password,
	)
	s.sellerClient.SetToken(sellerToken)

	s.adminClient = helpers.NewAPIClient(s.server)
	adminToken := helpers.Login(
		s.T(),
		s.adminClient,
		helpers.AdminEmail,
		helpers.AdminPassword,
	)
	s.adminClient.SetToken(adminToken)
}

func (s *OrderSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *OrderSuite) SetupTest() {
	s.cleanupOrderDomainData()
}

func TestOrderSuite(t *testing.T) {
	suite.Run(t, new(OrderSuite))
}

func (s *OrderSuite) cleanupOrderDomainData() {
	// Order graph cleanup (children first), then cart cleanup for test users.
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_history`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_item_applied_promotion`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_applied_coupon`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_applied_promotion`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_address`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_item`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM "order"`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM inventory_reservation`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM inventory_transaction`).Error)

	userIDs := []uint{
		helpers.CustomerUserID,
		helpers.Customer2UserID,
		helpers.Seller2UserID,
	}

	var cartIDs []uint
	s.Require().NoError(
		s.container.DB.Model(&orderEntity.Cart{}).
			Where("user_id IN ?", userIDs).
			Pluck("id", &cartIDs).
			Error,
	)
	if len(cartIDs) > 0 {
		var itemIDs []uint
		s.Require().NoError(
			s.container.DB.Model(&orderEntity.CartItem{}).
				Where("cart_id IN ?", cartIDs).
				Pluck("id", &itemIDs).Error,
		)
		if len(itemIDs) > 0 {
			s.Require().NoError(
				s.container.DB.Where("cart_item_id IN ?", itemIDs).
					Delete(&orderEntity.CartItemPromotion{}).Error,
			)
		}
		s.Require().NoError(
			s.container.DB.Where("cart_id IN ?", cartIDs).
				Delete(&orderEntity.CartAppliedCoupon{}).Error,
		)
		s.Require().NoError(
			s.container.DB.Where("cart_id IN ?", cartIDs).Delete(&orderEntity.CartItem{}).Error,
		)
	}
	s.Require().NoError(
		s.container.DB.Where("user_id IN ?", userIDs).Delete(&orderEntity.Cart{}).Error,
	)

	s.cleanupSeller2DiscountCodes()
}

func (s *OrderSuite) cleanupSeller2DiscountCodes() {
	var codeIDs []uint
	s.Require().NoError(
		s.container.DB.Table("discount_code").
			Where("seller_id = ?", helpers.Seller2UserID).
			Pluck("id", &codeIDs).Error,
	)
	if len(codeIDs) == 0 {
		return
	}
	s.Require().NoError(
		s.container.DB.Where("discount_code_id IN ?", codeIDs).
			Delete(&promotionEntity.DiscountCodeUsage{}).Error,
	)
	s.Require().NoError(
		s.container.DB.Where("discount_code_id IN ?", codeIDs).
			Delete(&promotionEntity.DiscountCodeProduct{}).Error,
	)
	s.Require().NoError(
		s.container.DB.Where("discount_code_id IN ?", codeIDs).
			Delete(&promotionEntity.DiscountCodeCategory{}).Error,
	)
	s.Require().NoError(
		s.container.DB.Where("discount_code_id IN ?", codeIDs).
			Delete(&promotionEntity.DiscountCodeCollection{}).Error,
	)
	s.Require().NoError(
		s.container.DB.Where("id IN ?", codeIDs).
			Delete(&promotionEntity.DiscountCode{}).Error,
	)
}
