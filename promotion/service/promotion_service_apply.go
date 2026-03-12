package service

import (
	"context"
	"sort"
	"time"

	"ecommerce-be/common/log"
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/factory"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/service/promotionStrategy"
)

// promotionCandidate holds a promotion and its calculated discount result
type promotionCandidate struct {
	promotion *entity.Promotion
	result    *model.PromotionValidationResult
}

// ApplyPromotionsToCart applies active seller promotions to a cart using priority + stacking rules.
//
// Core behavior:
// 1. Load active promotions and filter out invalid ones (date/status/usage/customer/scope conditions).
// 2. Group remaining promotions by priority (higher priority groups are processed first).
// 3. For each group, apply promotions sequentially via applyPriorityGroupSequentially:
//   - Re-evaluate all remaining promotions against the latest effective prices.
//   - Pick the current best discount candidate (greedy).
//   - Apply it immediately and update effective prices.
//   - Continue while stackable; stop when a non-stackable promotion is applied.
//
// 4. Carry forward effective prices from one priority group to the next.
// 5. If any applied promotion is non-stackable, stop processing further groups.
//
// Result:
// Returns AppliedPromotionSummary with applied/skipped promotions, per-item discount details,
// and final totals derived from the final effective prices.
func (s *PromotionServiceImpl) ApplyPromotionsToCart(
	ctx context.Context,
	cart *model.CartValidationRequest,
) (*model.AppliedPromotionSummary, error) {
	log.InfoWithContext(ctx, "Applying promotions to cart")

	effectivePrices := s.initEffectivePrices(cart)

	allPromotions, err := s.promotionRepo.FindActiveBySellerID(ctx, cart.SellerID)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to fetch active promotions", err)
		return nil, err
	}

	validPromotions, skippedResults := s.filterValidPromotions(ctx, allPromotions, cart)
	priorityGroups := s.groupByPriority(validPromotions)

	var appliedResults []model.PromotionValidationResult
	appliedCount := 0

	for _, group := range priorityGroups {
		applied, skipped := s.applyPriorityGroupSequentially(
			ctx,
			group,
			cart,
			effectivePrices,
			appliedCount,
		)
		appliedResults = append(appliedResults, applied...)
		skippedResults = append(skippedResults, skipped...)
		appliedCount += len(applied)

		// If a non-stackable promotion was applied, stop processing further groups.
		if s.hasAppliedNonStackable(applied) {
			break
		}
	}

	return s.buildPromotionSummary(cart, effectivePrices, appliedResults, skippedResults), nil
}

// applyPriorityGroupSequentially evaluates and applies promotions one-by-one in a priority group.
// Stackable promotions are re-evaluated on updated effective prices after each application.
func (s *PromotionServiceImpl) applyPriorityGroupSequentially(
	ctx context.Context,
	group []*entity.Promotion,
	cart *model.CartValidationRequest,
	effectivePrices map[string]int64,
	alreadyAppliedCount int,
) (applied []model.PromotionValidationResult, skipped []model.PromotionValidationResult) {
	remaining := append([]*entity.Promotion(nil), group...)

	for len(remaining) > 0 {
		candidates, stepSkipped := s.evaluateCandidates(ctx, remaining, cart, effectivePrices)
		skipped = append(skipped, stepSkipped...)

		// Remove promotions that were skipped in this step from remaining so they are reported once.
		if len(stepSkipped) > 0 {
			skippedIDs := make(map[uint]struct{}, len(stepSkipped))
			for _, s := range stepSkipped {
				if s.Promotion != nil {
					skippedIDs[s.Promotion.ID] = struct{}{}
				}
			}

			filtered := remaining[:0]
			for _, promo := range remaining {
				if _, wasSkipped := skippedIDs[promo.ID]; !wasSkipped {
					filtered = append(filtered, promo)
				}
			}
			remaining = filtered
		}

		if len(candidates) == 0 {
			break
		}

		best := candidates[0]
		canStack := best.promotion.CanStackWithOtherPromotions != nil &&
			*best.promotion.CanStackWithOtherPromotions

		// Non-stackable promotions cannot apply after any previous application.
		if (alreadyAppliedCount+len(applied)) > 0 && !canStack {
			skipped = append(
				skipped,
				skippedResult(best.promotion, "Cannot stack with other promotions"),
			)
			remaining = removePromotionByID(remaining, best.promotion.ID)
			continue
		}

		// Apply selected candidate and update effective prices immediately.
		for _, d := range best.result.ItemDiscounts {
			if d.FinalCents >= 0 {
				effectivePrices[d.ItemID] = d.FinalCents
			}
		}
		applied = append(applied, *best.result)
		remaining = removePromotionByID(remaining, best.promotion.ID)

		// A non-stackable promotion ends evaluation for this and subsequent groups.
		if !canStack {
			break
		}
	}

	return applied, skipped
}

func removePromotionByID(promotions []*entity.Promotion, id uint) []*entity.Promotion {
	filtered := promotions[:0]
	for _, promo := range promotions {
		if promo.ID != id {
			filtered = append(filtered, promo)
		}
	}
	return filtered
}

// initEffectivePrices builds the initial effective unit-price map from cart items.
func (s *PromotionServiceImpl) initEffectivePrices(
	cart *model.CartValidationRequest,
) map[string]int64 {
	prices := make(map[string]int64, len(cart.Items))
	for _, item := range cart.Items {
		prices[item.ItemID] = item.PriceCents
	}
	return prices
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

// evaluateCandidates calculates strategy discounts for a group and returns candidates sorted by
// total discount descending. Runtime is O(g * strategyCost + g log g), where g is group size.
func (s *PromotionServiceImpl) evaluateCandidates(
	ctx context.Context,
	group []*entity.Promotion,
	cart *model.CartValidationRequest,
	effectivePrices map[string]int64,
) ([]promotionCandidate, []model.PromotionValidationResult) {
	var candidates []promotionCandidate
	var skipped []model.PromotionValidationResult

	for _, promo := range group {
		strategy := promotionStrategy.GetPromotionStrategy(promo.PromotionType)
		if strategy == nil {
			skipped = append(skipped, skippedResult(promo, "Unsupported promotion type"))
			continue
		}

		result, err := strategy.CalculateDiscount(ctx, promo, cart, effectivePrices)
		if err != nil || result == nil || !result.IsValid {
			reason := "Promotion not applicable"
			if result != nil && result.Reason != "" {
				reason = result.Reason
			}
			skipped = append(skipped, skippedResult(promo, reason))
			continue
		}

		// Attach full promotion response (same as get-promotion API)
		result.Promotion = factory.PromotionEntityToResponse(promo)

		candidates = append(candidates, promotionCandidate{
			promotion: promo,
			result:    result,
		})
	}

	// Sort by total discount descending so best discount is applied first
	sort.Slice(candidates, func(a, b int) bool {
		totalA := candidates[a].result.DiscountCents + candidates[a].result.ShippingDiscount
		totalB := candidates[b].result.DiscountCents + candidates[b].result.ShippingDiscount
		return totalA > totalB
	})

	return candidates, skipped
}

// hasAppliedNonStackable checks whether any applied promotion is non-stackable.
func (s *PromotionServiceImpl) hasAppliedNonStackable(
	applied []model.PromotionValidationResult,
) bool {
	for _, a := range applied {
		if a.Promotion == nil {
			continue
		}
		if a.Promotion.CanStackWithOtherPromotions == nil ||
			!*a.Promotion.CanStackWithOtherPromotions {
			return true
		}
	}
	return false
}

// buildPromotionSummary builds the final summary with per-item breakdown
func (s *PromotionServiceImpl) buildPromotionSummary(
	cart *model.CartValidationRequest,
	effectivePrices map[string]int64,
	appliedResults []model.PromotionValidationResult,
	skippedResults []model.PromotionValidationResult,
) *model.AppliedPromotionSummary {
	var totalDiscount int64
	var shippingDiscount int64
	var originalSubtotal int64

	// Collect all item discounts per item, converting internal ItemDiscount → response ItemPromotionDetail
	itemPromotionsMap := make(map[string][]model.ItemPromotionDetail)
	for _, result := range appliedResults {
		totalDiscount += result.DiscountCents
		shippingDiscount += result.ShippingDiscount
		for _, id := range result.ItemDiscounts {
			itemPromotionsMap[id.ItemID] = append(
				itemPromotionsMap[id.ItemID],
				model.ItemPromotionDetail{
					PromotionID:   id.PromotionID,
					PromotionName: id.PromotionName,
					DiscountCents: id.DiscountCents,
					OriginalCents: id.OriginalCents,
					FinalCents:    id.FinalCents,
					FreeQuantity:  id.FreeQuantity,
				},
			)
		}
	}

	// Build per-item summaries
	items := make([]model.CartItemSummary, len(cart.Items))
	for i, item := range cart.Items {
		originalSubtotal += item.TotalCents
		finalPrice := effectivePrices[item.ItemID]
		itemTotalDiscount := (item.PriceCents - finalPrice) * int64(item.Quantity)

		items[i] = model.CartItemSummary{
			ItemID:             item.ItemID,
			ProductID:          item.ProductID,
			VariantID:          item.VariantID,
			Quantity:           item.Quantity,
			OriginalPriceCents: item.PriceCents,
			FinalPriceCents:    finalPrice,
			TotalDiscountCents: itemTotalDiscount,
			AppliedPromotions:  itemPromotionsMap[item.ItemID],
		}
	}

	finalSubtotal := originalSubtotal - totalDiscount
	if finalSubtotal < 0 {
		finalSubtotal = 0
	}

	return &model.AppliedPromotionSummary{
		Items:              items,
		AppliedPromotions:  appliedResults,
		SkippedPromotions:  skippedResults,
		TotalDiscountCents: totalDiscount,
		ShippingDiscount:   shippingDiscount,
		OriginalSubtotal:   originalSubtotal,
		FinalSubtotal:      finalSubtotal,
	}
}

// validateGeneralConditions checks all general promotion conditions before strategy-specific logic
func (s *PromotionServiceImpl) validateGeneralConditions(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
) string {
	if promotion.Status != entity.StatusActive {
		return "Promotion is not active"
	}

	now := time.Now()
	if promotion.StartsAt != nil && now.Before(*promotion.StartsAt) {
		return "Promotion has not started yet"
	}
	if promotion.EndsAt != nil && now.After(*promotion.EndsAt) {
		return "Promotion has ended"
	}

	// Fast-path filter: reject clearly exhausted promotions in memory.
	// Actual atomic enforcement happens at redemption time (order module).
	if promotion.UsageLimitTotal != nil &&
		promotion.CurrentUsageCount >= *promotion.UsageLimitTotal {
		return "Promotion usage limit reached"
	}

	// Per-customer usage limit check
	if promotion.UsageLimitPerCustomer != nil && cart.CustomerID != nil && *cart.CustomerID > 0 {
		userUsageCount, err := s.promotionRepo.CountUsageByUser(ctx, promotion.ID, *cart.CustomerID)
		if err != nil {
			log.ErrorWithContext(ctx, "Failed to check per-customer usage", err)
			return "Unable to verify customer usage limit"
		}
		if userUsageCount >= *promotion.UsageLimitPerCustomer {
			return "Customer usage limit reached for this promotion"
		}
	}

	if !s.isCustomerEligible(promotion, cart) {
		return "Customer is not eligible for this promotion"
	}

	if promotion.MinPurchaseAmountCents != nil &&
		cart.SubtotalCents < *promotion.MinPurchaseAmountCents {
		return "Minimum purchase amount not met"
	}

	if promotion.MinQuantity != nil {
		totalQuantity := 0
		for _, item := range cart.Items {
			totalQuantity += item.Quantity
		}
		if totalQuantity < *promotion.MinQuantity {
			return "Minimum quantity not met"
		}
	}

	if !s.isCartEligibleForScope(ctx, promotion, cart) {
		return "Cart items do not match promotion scope"
	}

	return ""
}

// isCartEligibleForScope checks if cart items match the promotion scope (AppliesTo)
func (s *PromotionServiceImpl) isCartEligibleForScope(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
) bool {
	switch promotion.AppliesTo {
	case entity.ScopeAllProducts:
		return true
	case entity.ScopeSpecificProducts:
		return s.isCartEligibleForProductScope(ctx, promotion.ID, cart)
	case entity.ScopeSpecificCategories:
		return s.isCartEligibleForCategoryScope(ctx, promotion.ID, cart)
	case entity.ScopeSpecificCollections:
		return s.isCartEligibleForCollectionScope(ctx, promotion.ID, cart)
	default:
		return false
	}
}

// isCartEligibleForProductScope checks if any cart item is in the promotion's product scope
func (s *PromotionServiceImpl) isCartEligibleForProductScope(
	ctx context.Context,
	promotionID uint,
	cart *model.CartValidationRequest,
) bool {
	// Collect product IDs from cart items
	cartProductIDs := make([]uint, len(cart.Items))
	for i, item := range cart.Items {
		cartProductIDs[i] = item.ProductID
	}

	// Call GetProducts with cart product IDs as filter
	// If any results come back, those cart products exist in the promotion scope
	resp, err := s.productScopeService.GetProducts(ctx, model.GetPromotionProductsRequest{
		GetPromotionScopeRequest: model.GetPromotionScopeRequest{
			BasePromotionScopeRequest: model.BasePromotionScopeRequest{PromotionID: promotionID},
		},
		ProductIDs: cartProductIDs,
	})
	if err != nil || resp == nil {
		return false
	}

	return len(resp.Products) > 0
}

// isCartEligibleForCategoryScope checks if any cart item's category is in the promotion's category scope
func (s *PromotionServiceImpl) isCartEligibleForCategoryScope(
	ctx context.Context,
	promotionID uint,
	cart *model.CartValidationRequest,
) bool {
	// Collect unique category IDs from cart items
	cartCategoryIDs := make([]uint, len(cart.Items))
	for i, item := range cart.Items {
		cartCategoryIDs[i] = item.CategoryID
	}

	// Call GetCategories with cart category IDs as filter
	resp, err := s.categoryScopeService.GetCategories(ctx, model.GetPromotionCategoriesRequest{
		GetPromotionScopeRequest: model.GetPromotionScopeRequest{
			BasePromotionScopeRequest: model.BasePromotionScopeRequest{PromotionID: promotionID},
		},
		CategoryIDs: cartCategoryIDs,
	})
	if err != nil || resp == nil {
		return false
	}

	return len(resp.Categories) > 0
}

// isCartEligibleForCollectionScope checks if any cart item's product belongs to the promotion's collection scope
// Flow: promotion → collection IDs → product IDs (via product service) → match cart items
func (s *PromotionServiceImpl) isCartEligibleForCollectionScope(
	ctx context.Context,
	promotionID uint,
	cart *model.CartValidationRequest,
) bool {
	// Step 1: Get collections for this promotion
	collResp, err := s.collectionScopeService.GetCollections(
		ctx,
		model.GetPromotionCollectionsRequest{
			GetPromotionScopeRequest: model.GetPromotionScopeRequest{
				BasePromotionScopeRequest: model.BasePromotionScopeRequest{
					PromotionID: promotionID,
				},
			},
		},
	)
	if err != nil || collResp == nil || len(collResp.Collections) == 0 {
		return false
	}

	// Step 2: Extract collection IDs
	collectionIDs := make([]uint, len(collResp.Collections))
	for i, c := range collResp.Collections {
		collectionIDs[i] = c.CollectionID
	}

	// Step 3: Get product IDs belonging to those collections (via product service)
	productIDs, err := s.collectionProductService.GetProductIDsByCollectionIDs(ctx, collectionIDs)
	if err != nil || len(productIDs) == 0 {
		return false
	}

	// Step 4: Check if any cart item matches
	eligibleProducts := make(map[uint]bool, len(productIDs))
	for _, pid := range productIDs {
		eligibleProducts[pid] = true
	}

	for _, item := range cart.Items {
		if eligibleProducts[item.ProductID] {
			return true
		}
	}
	return false
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

// skippedResult creates a PromotionValidationResult for a skipped promotion
// with full promotion response details (same format as applied promotions).
func skippedResult(promo *entity.Promotion, reason string) model.PromotionValidationResult {
	return model.PromotionValidationResult{
		Promotion: factory.PromotionEntityToResponse(promo),
		Reason:    reason,
	}
}
