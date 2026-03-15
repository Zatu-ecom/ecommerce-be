package service

import (
	"context"
	"sort"
	"strconv"

	"ecommerce-be/common/log"
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/factory"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/service/promotionStrategy"
)

type PromoResult struct {
	promotion *entity.Promotion
	result    model.AppliedPromotionSummary
}

// ApplyPromotionsToCart applies active seller promotions to a cart using priority + stacking rules.
//
// Core behavior:
//
//  1. Load active promotions and filter out invalid ones (date/status/usage/customer/scope conditions).
//
//  2. Group remaining promotions by priority (higher priority groups are processed first).
//
//  3. For each group, apply promotions sequentially via applyPriorityGroupSequentially:
//     - Re-evaluate all remaining promotions against the latest effective prices.
//     - Pick the current best discount candidate (greedy).
//     - Apply it immediately and update effective prices.
//     - Continue while stackable; stop when a non-stackable promotion is applied.
//
//  4. Carry forward effective prices from one priority group to the next.
//
//  5. If any applied promotion is non-stackable, stop processing further groups.
//
// Result:
// Returns AppliedPromotionSummary with applied/skipped promotions, per-item discount details,
// and final totals derived from the final effective prices.
func (s *PromotionServiceImpl) ApplyPromotionsToCartV2(
	ctx context.Context,
	cart *model.CartValidationRequest,
) (*model.AppliedPromotionSummary, error) {
	log.InfoWithContext(ctx, "Applying promotions to cart")
	allPromotions, err := s.promotionRepo.FindActiveBySellerID(ctx, cart.SellerID)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to fetch active promotions", err)
		return nil, err
	}

	result := constructAppliedPromotionSummaryFromCartRequest(cart)
	err = s.applyPromotionBasedOnPriority(ctx, result, allPromotions, cart)

	return result, err
}

func (s *PromotionServiceImpl) applyPromotionBasedOnPriority(
	ctx context.Context,
	summary *model.AppliedPromotionSummary,
	promotions []*entity.Promotion,
	cart *model.CartValidationRequest,
) error {
	validPromotions, skippedResults := s.filterValidPromotions(ctx, promotions, cart)
	priorityGroups := s.groupByPriority(validPromotions)
	promoIdVslineItems := make(map[uint][]string)

	summary.SkippedPromotions = append(summary.SkippedPromotions, skippedResults...)

	// Now we will create a copy of the summary as in first time we find best promotion,
	// we will update the summary in-place and next time when we want to check eligibility of next promotion,
	// we want to check based on updated summary instead of original cart request
	for _, group := range priorityGroups {
		// For each priority group, we will find the best promotion
		sortedResults := s.findBestPromotionForCart(ctx, summary, group, cart, promoIdVslineItems)

		// Now all promotions in this priority group will be applied sequentially
		for _, promoResult := range sortedResults {
			promo := promoResult.promotion

			eligibleLineItems := promoIdVslineItems[promo.ID]
			promotionStrategy := promotionStrategy.GetPromotionStrategy(promo.PromotionType)

			if err := promotionStrategy.CalculateDiscountV2(ctx, promo, cart, summary, eligibleLineItems); err != nil {
				handlePromotionCalculationError(ctx, summary, promo, err)
				continue
			}

			if !*promo.CanStackWithOtherPromotions && len(summary.AppliedPromotions) > 0 {
				log.InfoWithContext(
					ctx,
					"Non-stackable promotion "+promo.Name+" cannot be applied as another promotion is already applied",
				)
				summary.SkippedPromotions = append(
					summary.SkippedPromotions,
					model.PromotionValidationResult{
						Promotion: factory.PromotionEntityToResponse(promo),
						IsValid:   false,
						Reason:    "Non-stackable promotion cannot be applied as another promotion is already applied",
					},
				)

				return nil
			}

		}
	}

	return nil
}

func (s *PromotionServiceImpl) findBestPromotionForCart(
	ctx context.Context,
	summary *model.AppliedPromotionSummary,
	promotions []*entity.Promotion,
	cart *model.CartValidationRequest,
	promoIdVslineItems map[uint][]string,
) []PromoResult {
	promoResults := make([]PromoResult, 0, len(promotions))
	for _, promo := range promotions {
		var eligibleLineItems []string

		scopeService := s.promotionScopeEligibilityServiceFactory.
			GetPromotionScopeEligibilityService(promo.AppliesTo)

		if scopeService != nil {
			_, eligibleLineItems = scopeService.IsCartEligible(ctx, promo.ID, cart)
		} else {
			eligibleLineItems = make([]string, len(cart.Items))
			for i, item := range cart.Items {
				eligibleLineItems[i] = item.ItemID
			}
		}

		promoIdVslineItems[promo.ID] = eligibleLineItems

		summaryCopy := deepCopySummary(summary)
		promotionStrategy := promotionStrategy.GetPromotionStrategy(promo.PromotionType)

		if err := promotionStrategy.CalculateDiscountV2(ctx, promo, cart, summaryCopy, eligibleLineItems); err != nil {
			handlePromotionCalculationError(ctx, summary, promo, err)
			continue
		}
		promoResults = append(promoResults, PromoResult{
			promotion: promo,
			result:    *summaryCopy,
		})
	}

	// Sort by total discount descending so best discount is applied first
	sort.Slice(promoResults, func(a, b int) bool {
		totalA := promoResults[a].result.TotalDiscountCents + promoResults[a].result.ShippingDiscount
		totalB := promoResults[b].result.TotalDiscountCents + promoResults[b].result.ShippingDiscount
		return totalA > totalB
	})

	return promoResults
}

func handlePromotionCalculationError(
	ctx context.Context,
	summary *model.AppliedPromotionSummary,
	promo *entity.Promotion,
	err error,
) {
	log.ErrorWithContext(
		ctx,
		"Failed to calculate discount for promotion "+strconv.FormatUint(
			uint64(promo.ID),
			10,
		),
		err,
	)
	// because of error we are skipping this promotion but we are not returning error as we want to continue applying
	// other promotions and not fail the entire flow because of one promotion failure
	summary.SkippedPromotions = append(
		summary.SkippedPromotions,
		model.PromotionValidationResult{
			Promotion: factory.PromotionEntityToResponse(promo),
			IsValid:   false,
			Reason:    err.Error(),
		},
	)
}

// deepCopySummary returns a fully independent copy of the summary so that
// trial V2 calls during ranking do not leak item-level mutations back to
// the original summary.
func deepCopySummary(src *model.AppliedPromotionSummary) *model.AppliedPromotionSummary {
	dst := *src

	dst.Items = make([]model.CartItemSummary, len(src.Items))
	for i, item := range src.Items {
		dst.Items[i] = item
		dst.Items[i].AppliedPromotions = make([]model.ItemPromotionDetail, len(item.AppliedPromotions))
		copy(dst.Items[i].AppliedPromotions, item.AppliedPromotions)
	}

	dst.AppliedPromotions = make([]model.PromotionValidationResult, len(src.AppliedPromotions))
	copy(dst.AppliedPromotions, src.AppliedPromotions)

	dst.SkippedPromotions = make([]model.PromotionValidationResult, len(src.SkippedPromotions))
	copy(dst.SkippedPromotions, src.SkippedPromotions)

	return &dst
}

func constructAppliedPromotionSummaryFromCartRequest(
	cart *model.CartValidationRequest,
) *model.AppliedPromotionSummary {
	items := make([]model.CartItemSummary, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = model.CartItemSummary{
			ItemID:             item.ItemID,
			ProductID:          item.ProductID,
			VariantID:          item.VariantID,
			Quantity:           item.Quantity,
			OriginalPriceCents: item.PriceCents,
			FinalPriceCents:    item.TotalCents, // Initial final price is same as original; will be reduced as promotions are applied
			TotalDiscountCents: 0,
			AppliedPromotions:  []model.ItemPromotionDetail{}, // Initial total discount is 0; will be updated as promotions are applied
		}
	}

	return &model.AppliedPromotionSummary{
		Items:              items,
		AppliedPromotions:  []model.PromotionValidationResult{},
		SkippedPromotions:  []model.PromotionValidationResult{},
		ShippingDiscount:   0,
		OriginalSubtotal:   cart.SubtotalCents,
		FinalSubtotal:      cart.SubtotalCents, // Initial final subtotal is same as original; will be reduced as promotions are applied
		TotalDiscountCents: 0,                  // Initial final total is subtotal + shipping; will be reduced as promotions are applied
	}
}
