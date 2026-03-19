package service

import (
	"context"
	"sort"
	"strconv"
	"time"

	"ecommerce-be/common/log"
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/factory"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/service/promotionStrategy"
	promotionConstant "ecommerce-be/promotion/utils/constant"
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
func (s *PromotionServiceImpl) ApplyPromotionsToCart(
	ctx context.Context,
	cart *model.CartValidationRequest,
) (*model.AppliedPromotionSummary, error) {
	log.InfoWithContext(ctx, "Applying promotions to cart")
	allPromotions, err := s.promotionRepo.FindActiveBySellerID(ctx, cart.SellerID)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to fetch active promotions", err)
		return nil, err
	}

	result := factory.ConstructAppliedPromotionSummaryFromCartRequest(cart)
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

			if err := promotionStrategy.CalculateDiscount(ctx, promo, cart, summary, eligibleLineItems); err != nil {
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
						Reason:    promotionConstant.VALIDATION_NON_STACKABLE_ALREADY_APPLIED_MSG,
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

		if err := promotionStrategy.CalculateDiscount(ctx, promo, cart, summaryCopy, eligibleLineItems); err != nil {
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

// filterValidPromotions validates general conditions and splits promotions into valid vs skipped.
// Runtime is linear in number of promotions plus any repository/service checks inside validations.
func (s *PromotionServiceImpl) filterValidPromotions(
	ctx context.Context,
	promotions []*entity.Promotion,
	cart *model.CartValidationRequest,
) ([]*entity.Promotion, []model.PromotionValidationResult) {
	var valid []*entity.Promotion
	var skipped []model.PromotionValidationResult

	for _, promo := range promotions {
		if reason := s.validateGeneralConditions(ctx, promo, cart); reason != "" {
			skipped = append(skipped, skippedResult(promo, reason))
			continue
		}
		valid = append(valid, promo)
	}

	return valid, skipped
}

// validateGeneralConditions checks all general promotion conditions before strategy-specific logic
func (s *PromotionServiceImpl) validateGeneralConditions(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
) string {
	if promotion.Status != entity.StatusActive {
		return promotionConstant.VALIDATION_PROMOTION_NOT_ACTIVE_MSG
	}

	now := time.Now()
	if promotion.StartsAt != nil && now.Before(*promotion.StartsAt) {
		return promotionConstant.VALIDATION_PROMOTION_NOT_STARTED_MSG
	}
	if promotion.EndsAt != nil && now.After(*promotion.EndsAt) {
		return promotionConstant.VALIDATION_PROMOTION_ENDED_MSG
	}

	// Fast-path filter: reject clearly exhausted promotions in memory.
	// Actual atomic enforcement happens at redemption time (order module).
	if promotion.UsageLimitTotal != nil &&
		promotion.CurrentUsageCount >= *promotion.UsageLimitTotal {
		return promotionConstant.VALIDATION_PROMOTION_USAGE_LIMIT_REACHED_MSG
	}

	// Per-customer usage limit check
	if promotion.UsageLimitPerCustomer != nil && cart.CustomerID != nil && *cart.CustomerID > 0 {
		userUsageCount, err := s.promotionRepo.CountUsageByUser(ctx, promotion.ID, *cart.CustomerID)
		if err != nil {
			log.ErrorWithContext(ctx, "Failed to check per-customer usage", err)
			return promotionConstant.VALIDATION_UNABLE_TO_VERIFY_CUSTOMER_USAGE_MSG
		}
		if userUsageCount >= *promotion.UsageLimitPerCustomer {
			return promotionConstant.VALIDATION_CUSTOMER_USAGE_LIMIT_REACHED_MSG
		}
	}

	if !s.isCustomerEligible(promotion, cart) {
		return promotionConstant.VALIDATION_CUSTOMER_NOT_ELIGIBLE_MSG
	}

	return ""
}

// isCustomerEligible checks if customer is eligible for the promotion
func (s *PromotionServiceImpl) isCustomerEligible(
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
) bool {
	switch promotion.EligibleFor {
	case entity.EligibleEveryone:
		return true
	case entity.EligibleNewCustomers:
		// New customer = first-time buyer (IsFirstOrder == true)
		return cart.IsFirstOrder
	case entity.EligibleSpecificSegment:
		if promotion.CustomerSegmentID == nil {
			return false
		}
		// DEFERRED: Customer segment matching — see PROMOTION_DEFERRED.md
		return false
	default:
		return false
	}
}

// groupByPriority groups pre-sorted promotions by priority in a single pass.
func (s *PromotionServiceImpl) groupByPriority(
	promotions []*entity.Promotion,
) [][]*entity.Promotion {
	if len(promotions) == 0 {
		return nil
	}

	var groups [][]*entity.Promotion
	var currentGroup []*entity.Promotion
	currentPriority := promotions[0].Priority

	for _, promo := range promotions {
		if promo.Priority != currentPriority {
			groups = append(groups, currentGroup)
			currentGroup = nil
			currentPriority = promo.Priority
		}
		currentGroup = append(currentGroup, promo)
	}
	groups = append(groups, currentGroup)

	return groups
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
	// because of error we are skipping this promotion but we are not returning error
	// as we want to continue applying other promotions and not fail the entire flow
	// because of one promotion failure
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
// trial calls during ranking do not leak item-level mutations back to
// the original summary.
func deepCopySummary(
	src *model.AppliedPromotionSummary,
) *model.AppliedPromotionSummary {
	dst := *src

	dst.Items = make([]model.CartItemSummary, len(src.Items))
	for i, item := range src.Items {
		dst.Items[i] = item
		dst.Items[i].AppliedPromotions = make(
			[]model.ItemPromotionDetail,
			len(item.AppliedPromotions),
		)
		copy(dst.Items[i].AppliedPromotions, item.AppliedPromotions)
	}

	dst.AppliedPromotions = make([]model.PromotionValidationResult, len(src.AppliedPromotions))
	copy(dst.AppliedPromotions, src.AppliedPromotions)

	dst.SkippedPromotions = make([]model.PromotionValidationResult, len(src.SkippedPromotions))
	copy(dst.SkippedPromotions, src.SkippedPromotions)

	return &dst
}

// skippedResult creates a PromotionValidationResult for a skipped promotion
// with full promotion response details (same format as applied promotions).
func skippedResult(promo *entity.Promotion, reason string) model.PromotionValidationResult {
	return model.PromotionValidationResult{
		Promotion: factory.PromotionEntityToResponse(promo),
		Reason:    reason,
	}
}
