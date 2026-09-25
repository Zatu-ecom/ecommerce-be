package discountCodeStrategy

import (
	"context"

	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
)

type FreeShippingDiscountStrategy struct{}

func (s *FreeShippingDiscountStrategy) Calculate(
	_ context.Context,
	_ *entity.DiscountCode,
	req *model.CouponCartRequest,
	_ []string,
) (int64, int64, error) {
	if req.ShippingCents <= 0 {
		return 0, 0, nil
	}
	return 0, req.ShippingCents, nil
}
