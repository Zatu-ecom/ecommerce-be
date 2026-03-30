package promotionStrategy

import (
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/factory"
	"ecommerce-be/promotion/model"
)

// ItemDiscountDetail is a lightweight per-item discount result produced by a strategy
// before it is merged into the AppliedPromotionSummary.
type ItemDiscountDetail struct {
	ItemID        string
	DiscountCents int64
	FreeQuantity  int
}

// ApplyDiscountToSummary merges a strategy's computed discounts into the shared
// AppliedPromotionSummary in-place. It is designed to be called by every strategy's
// CalculateDiscount so that the book-keeping logic is not duplicated.
//
// FinalPriceCents on CartItemSummary is a line total (PriceCents * Quantity), and
// ItemDiscountDetail.DiscountCents is likewise a line-level amount, so no per-unit
// conversion is performed here.
//
// For each affected item it:
//   - subtracts DiscountCents (line total) from FinalPriceCents (line total)
//   - accumulates TotalDiscountCents on the CartItemSummary
//   - appends an ItemPromotionDetail to the item's AppliedPromotions list
//
// Then at the summary level it:
//   - appends a PromotionValidationResult to AppliedPromotions
//   - adds totalDiscountCents to summary.TotalDiscountCents
//   - subtracts totalDiscountCents from summary.FinalSubtotal
func ApplyDiscountToSummary(
	summary *model.AppliedPromotionSummary,
	promotion *entity.Promotion,
	itemDiscounts []ItemDiscountDetail,
	totalDiscountCents int64,
	shippingDiscount int64,
) {
	// Build a quick lookup so we don't iterate items slice for every discount entry
	itemIndexByID := make(map[string]int, len(summary.Items))
	for i, item := range summary.Items {
		itemIndexByID[item.ItemID] = i
	}

	for _, d := range itemDiscounts {
		idx, ok := itemIndexByID[d.ItemID]
		if !ok {
			continue
		}
		item := &summary.Items[idx]

		// Both FinalPriceCents and DiscountCents are line totals; subtract directly.
		newFinal := item.FinalPriceCents - d.DiscountCents
		if newFinal < 0 {
			newFinal = 0
		}

		item.AppliedPromotions = append(item.AppliedPromotions, model.ItemPromotionDetail{
			PromotionID:   promotion.ID,
			PromotionName: promotion.Name,
			DiscountCents: d.DiscountCents,
			OriginalCents: item.FinalPriceCents, // line total before this promotion
			FinalCents:    newFinal,             // line total after this promotion
			FreeQuantity:  d.FreeQuantity,
		})

		item.FinalPriceCents = newFinal
		item.TotalDiscountCents += d.DiscountCents
	}

	// Append to the top-level applied promotions list (do not replace — stacking is supported)
	summary.AppliedPromotions = append(summary.AppliedPromotions, model.PromotionValidationResult{
		Promotion:        factory.PromotionEntityToResponse(promotion),
		IsValid:          true,
		DiscountCents:    totalDiscountCents,
		ShippingDiscount: shippingDiscount,
	})

	summary.TotalDiscountCents += totalDiscountCents
	summary.ShippingDiscount += shippingDiscount
	summary.FinalSubtotal -= totalDiscountCents
	if summary.FinalSubtotal < 0 {
		summary.FinalSubtotal = 0
	}
}
