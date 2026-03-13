package promotion_test

import (
	"context"
	"strconv"
	"testing"

	"ecommerce-be/common/helper"
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/service/promotionStrategy"
)

func BenchmarkBuyXGetYStrategySameRewardLargeQuantity(b *testing.B) {
	strategy := promotionStrategy.NewBuyXGetYStrategy()
	promotion := &entity.Promotion{
		Name:      "BxGy Same Category",
		AppliesTo: entity.ScopeAllProducts,
		DiscountConfig: map[string]interface{}{
			"buy_quantity":   2,
			"get_quantity":   1,
			"is_same_reward": true,
			"scope_type":     string(model.BuyXGetYScopeSameCategory),
		},
	}

	cartItems := make([]model.CartItem, 0, 200)
	variantID := uint(1000)
	for i := 0; i < 200; i++ {
		variantID++
		price := int64(3000 + (i%25)*250)
		cartItems = append(cartItems, model.CartItem{
			ItemID:     "same-cat-item-" + strconv.Itoa(i),
			ProductID:  uint(100 + i),
			VariantID:  helper.UintPtr(variantID),
			CategoryID: 4,
			Quantity:   10,
			PriceCents: price,
			TotalCents: price * 10,
		})
	}

	effectivePrices := benchmarkEffectivePrices(cartItems)
	cart := benchmarkCart(cartItems)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := strategy.CalculateDiscount(
			context.Background(),
			promotion,
			cart,
			effectivePrices,
		)
		if err != nil {
			b.Fatalf("unexpected benchmark error: %v", err)
		}
		if result == nil || !result.IsValid {
			b.Fatalf("expected valid same-reward benchmark result")
		}
	}
}

func BenchmarkBuyXGetYStrategyCrossRewardLargeQuantity(b *testing.B) {
	strategy := promotionStrategy.NewBuyXGetYStrategy()
	promotion := &entity.Promotion{
		Name:      "Buy Phone Get Headphones",
		AppliesTo: entity.ScopeAllProducts,
		DiscountConfig: map[string]interface{}{
			"buy_quantity":   1,
			"get_quantity":   1,
			"is_same_reward": false,
			"get_product_id": 9001,
		},
	}

	cartItems := make([]model.CartItem, 0, 251)
	variantID := uint(2000)
	for i := 0; i < 250; i++ {
		variantID++
		price := int64(5000 + (i%30)*200)
		cartItems = append(cartItems, model.CartItem{
			ItemID:     "buy-item-" + strconv.Itoa(i),
			ProductID:  uint(200 + i),
			VariantID:  helper.UintPtr(variantID),
			CategoryID: 4,
			Quantity:   4,
			PriceCents: price,
			TotalCents: price * 4,
		})
	}
	cartItems = append(cartItems, model.CartItem{
		ItemID:     "reward-line-1",
		ProductID:  9001,
		VariantID:  helper.UintPtr(99901),
		CategoryID: 4,
		Quantity:   300,
		PriceCents: 1999,
		TotalCents: 1999 * 300,
	})

	effectivePrices := benchmarkEffectivePrices(cartItems)
	cart := benchmarkCart(cartItems)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := strategy.CalculateDiscount(
			context.Background(),
			promotion,
			cart,
			effectivePrices,
		)
		if err != nil {
			b.Fatalf("unexpected benchmark error: %v", err)
		}
		if result == nil || !result.IsValid {
			b.Fatalf("expected valid cross-reward benchmark result")
		}
	}
}

func benchmarkEffectivePrices(items []model.CartItem) map[string]int64 {
	effective := make(map[string]int64, len(items))
	for _, item := range items {
		effective[item.ItemID] = item.PriceCents
	}
	return effective
}

func benchmarkCart(items []model.CartItem) *model.CartValidationRequest {
	return &model.CartValidationRequest{
		SellerID:      1,
		Items:         items,
		SubtotalCents: subtotal(items),
	}
}

func subtotal(items []model.CartItem) int64 {
	var total int64
	for _, item := range items {
		total += item.TotalCents
	}
	return total
}
