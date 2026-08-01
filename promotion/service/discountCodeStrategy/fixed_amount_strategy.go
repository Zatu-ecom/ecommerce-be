package discountCodeStrategy

import (
	"context"

	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
)

type FixedAmountDiscountStrategy struct{}

func (s *FixedAmountDiscountStrategy) Calculate(
	_ context.Context,
	code *entity.DiscountCode,
	req *model.CouponCartRequest,
	eligibleItemIDs []string,
) (int64, int64, error) {
	subtotal := eligibleSubtotal(req, eligibleItemIDs)
	if subtotal <= 0 || code.Value <= 0 {
		return 0, 0, nil
	}
	discount := code.Value
	if discount > subtotal {
		discount = subtotal
	}
	return discount, 0, nil
}
