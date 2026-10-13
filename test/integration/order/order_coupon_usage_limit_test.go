package order_test

import (
	"net/http"
	"sync"
	"time"

	promotionEntity "ecommerce-be/promotion/entity"
	"ecommerce-be/test/integration/helpers"
)

func (s *OrderSuite) TestGlobalUsageLimitBlocksSecondCheckout() {
	s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("GLOBLIM1").Percentage(10).UsageLimitTotal(1).Build(),
	)

	s.addItemToCart(1, 1)
	s.applyCoupon("GLOBLIM1")
	s.createOrderOK()

	s.addItemToCart(1, 1)
	res := s.customerClient.Post(s.T(), "/api/order/cart/coupon", helpers.ApplyCouponPayload("GLOBLIM1"))
	s.Require().Equal(http.StatusBadRequest, res.Code, res.Body.String())
	errResp := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("COUPON_USAGE_LIMIT_REACHED", errResp["code"])
}

func (s *OrderSuite) TestPerCustomerUsageLimitBlocksWithinResetWindow() {
	s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("PERCUST1").
			Percentage(10).
			UsageLimitPerCustomer(1).
			UsageReset("day", 7).
			Build(),
	)

	s.addItemToCart(1, 1)
	s.applyCoupon("PERCUST1")
	s.createOrderOK()

	s.addItemToCart(1, 1)
	res := s.customerClient.Post(s.T(), "/api/order/cart/coupon", helpers.ApplyCouponPayload("PERCUST1"))
	s.Require().Equal(http.StatusBadRequest, res.Code, res.Body.String())
	errResp := helpers.AssertErrorResponse(s.T(), res, http.StatusBadRequest)
	s.Equal("COUPON_ALREADY_USED", errResp["code"])
}

func (s *OrderSuite) TestPerCustomerUsageAllowsAfterResetWindow() {
	codeID := s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("PERRESET").
			Percentage(10).
			UsageLimitPerCustomer(1).
			UsageReset("day", 1).
			Build(),
	)

	s.addItemToCart(1, 1)
	s.applyCoupon("PERRESET")
	s.createOrderOK()

	// Backdate usage outside the 1-day window
	past := time.Now().UTC().Add(-48 * time.Hour)
	s.Require().NoError(
		s.container.DB.Model(&promotionEntity.DiscountCodeUsage{}).
			Where("discount_code_id = ?", codeID).
			Update("used_at", past).Error,
	)

	s.addItemToCart(1, 1)
	res := s.customerClient.Post(s.T(), "/api/order/cart/coupon", helpers.ApplyCouponPayload("PERRESET"))
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())
}

func (s *OrderSuite) TestRaceSafeGlobalUsageLimit() {
	codeID := s.createSellerDiscountCode(
		helpers.NewDiscountCodePayload("RACELIM1").Percentage(10).UsageLimitTotal(1).Build(),
	)

	// Temporarily put customer2 on seller 2 so both can redeem the same storefront code.
	s.Require().NoError(
		s.container.DB.Exec(`UPDATE "user" SET seller_id = 2 WHERE id = ?`, helpers.Customer2UserID).Error,
	)
	defer func() {
		_ = s.container.DB.Exec(`UPDATE "user" SET seller_id = 3 WHERE id = ?`, helpers.Customer2UserID).Error
	}()

	customer2 := helpers.NewAPIClient(s.server)
	customer2.SetToken(helpers.Login(s.T(), customer2, helpers.Customer2Email, helpers.Customer2Password))

	// Prepare carts with the coupon for both customers before racing checkout.
	s.addItemToCart(1, 1)
	s.applyCoupon("RACELIM1")

	w := customer2.Post(s.T(), "/api/order/cart/item", helpers.AddCartItemsPayload(2, 1))
	s.Require().Equal(http.StatusCreated, w.Code, w.Body.String())
	w = customer2.Post(s.T(), "/api/order/cart/coupon", helpers.ApplyCouponPayload("RACELIM1"))
	s.Require().Equal(http.StatusOK, w.Code, w.Body.String())

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		statuses []int
	)
	place := func(client *helpers.APIClient, shippingAddressID uint) {
		defer wg.Done()
		req := helpers.NewCreateOrderRequest().
			ShippingAddressID(shippingAddressID).
			BillingAddressID(shippingAddressID).
			Build()
		res := client.Post(s.T(), OrderAPIEndpoint, req)
		mu.Lock()
		statuses = append(statuses, res.Code)
		mu.Unlock()
	}

	wg.Add(2)
	go place(s.customerClient, 1)
	go place(customer2, 3)
	wg.Wait()

	success, fail := 0, 0
	for _, code := range statuses {
		switch code {
		case http.StatusCreated:
			success++
		default:
			fail++
		}
	}
	s.Equal(1, success, "exactly one checkout should win the usage race: %v", statuses)
	s.Equal(1, fail, "exactly one checkout should lose the usage race: %v", statuses)

	var dc promotionEntity.DiscountCode
	s.Require().NoError(s.container.DB.First(&dc, codeID).Error)
	s.Equal(1, dc.CurrentUsageCount)

	var usageCount int64
	s.Require().NoError(
		s.container.DB.Model(&promotionEntity.DiscountCodeUsage{}).
			Where("discount_code_id = ?", codeID).Count(&usageCount).Error,
	)
	s.Equal(int64(1), usageCount)
}
