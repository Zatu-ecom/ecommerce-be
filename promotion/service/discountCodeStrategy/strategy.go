package discountCodeStrategy

import (
	"context"

	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
)

// DiscountStrategy calculates merchandise and shipping discounts for a coupon.
type DiscountStrategy interface {
	Calculate(
		ctx context.Context,
		code *entity.DiscountCode,
		req *model.CouponCartRequest,
		eligibleItemIDs []string,
	) (merchandiseDiscountCents int64, shippingDiscountCents int64, err error)
}

// GetDiscountStrategy returns the calculator for a discount type.
func GetDiscountStrategy(t entity.DiscountType) DiscountStrategy {
	switch t {
	case entity.DiscountPercentage:
		return &PercentageDiscountStrategy{}
	case entity.DiscountFixedAmount:
		return &FixedAmountDiscountStrategy{}
	case entity.DiscountFreeShipping:
		return &FreeShippingDiscountStrategy{}
	case entity.DiscountBuyXGetY:
		return &BuyXGetYDiscountStrategy{}
	default:
		return nil
	}
}

func eligibleSubtotal(req *model.CouponCartRequest, eligibleItemIDs []string) int64 {
	set := map[string]struct{}{}
	for _, id := range eligibleItemIDs {
		set[id] = struct{}{}
	}
	var total int64
	for _, item := range req.Items {
		if _, ok := set[item.ItemID]; ok {
			total += item.TotalCents
		}
	}
	return total
}

func eligibleItems(req *model.CouponCartRequest, eligibleItemIDs []string) []model.CartItem {
	set := map[string]struct{}{}
	for _, id := range eligibleItemIDs {
		set[id] = struct{}{}
	}
	out := make([]model.CartItem, 0)
	for _, item := range req.Items {
		if _, ok := set[item.ItemID]; ok {
			out = append(out, item)
		}
	}
	return out
}
