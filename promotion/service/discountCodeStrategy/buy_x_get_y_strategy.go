package discountCodeStrategy

import (
	"context"
	"sort"

	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
)

type BuyXGetYDiscountStrategy struct{}

func (s *BuyXGetYDiscountStrategy) Calculate(
	_ context.Context,
	code *entity.DiscountCode,
	req *model.CouponCartRequest,
	eligibleItemIDs []string,
) (int64, int64, error) {
	buyQty := intFromMeta(code.Metadata, "buyQuantity")
	getQty := intFromMeta(code.Metadata, "getQuantity")
	getPct := floatFromMeta(code.Metadata, "getDiscountPercent")
	if buyQty < 1 || getQty < 1 || getPct < 0 || getPct > 100 {
		return 0, 0, nil
	}

	items := eligibleItems(req, eligibleItemIDs)
	type unit struct {
		priceCents int64
	}
	units := make([]unit, 0)
	for _, item := range items {
		for i := 0; i < item.Quantity; i++ {
			units = append(units, unit{priceCents: item.PriceCents})
		}
	}
	if len(units) < buyQty+getQty {
		return 0, 0, nil
	}

	sort.Slice(units, func(i, j int) bool {
		return units[i].priceCents < units[j].priceCents
	})

	sets := len(units) / (buyQty + getQty)
	if sets == 0 {
		return 0, 0, nil
	}

	var discount int64
	// Cheapest units are the "get" items
	for set := 0; set < sets; set++ {
		for g := 0; g < getQty; g++ {
			idx := set*(buyQty+getQty) + g
			if idx >= len(units) {
				break
			}
			discount += int64(float64(units[idx].priceCents) * getPct / 100.0)
		}
	}
	return discount, 0, nil
}

func intFromMeta(meta map[string]any, key string) int {
	v, ok := meta[key]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	default:
		return 0
	}
}

func floatFromMeta(meta map[string]any, key string) float64 {
	v, ok := meta[key]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return 0
	}
}
