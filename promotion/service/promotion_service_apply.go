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

// ApplyPromotionsToCart applies all active promotions for the given seller to the cart.
//
// Flow:
//  1. Fetches all active promotions for the seller from the database (sorted by priority DESC).
//  2. Filters out promotions that fail general validation (status, date range, usage limits,
//     customer eligibility, minimum purchase/quantity, and scope matching).
//  3. Groups the remaining valid promotions by priority level.
//  4. For each priority group (highest priority first):
//     a. Evaluates every promotion in the group by running its strategy-specific
//     CalculateDiscount logic against the cart's current effective prices.
//     b. Sorts evaluated candidates by total discount descending (best discount first).
//     c. Applies candidates respecting stacking rules:
//     - If a promotion has CanStackWithOtherPromotions=true, it is applied and the
//     effective prices are updated so the next promotion calculates on the discounted price.
//     - If a promotion has CanStackWithOtherPromotions=false and no other promotion
//     has been applied yet, it is applied and no further promotions are processed.
//     - If a promotion has CanStackWithOtherPromotions=false but other promotions
//     are already applied, it is skipped.
//  5. Builds and returns an AppliedPromotionSummary containing:
//     - Per-item discount breakdown (which promotions affected each item and by how much).
//     - List of applied promotions with their discount details.
//     - List of skipped promotions with reasons for skipping.
//     - Totals: original subtotal, final subtotal, total discount, and shipping discount.
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
		candidates, skipped := s.evaluateCandidates(ctx, group, cart, effectivePrices)
		skippedResults = append(skippedResults, skipped...)

		applied, skipped := s.applyBestCandidates(candidates, effectivePrices, appliedCount)
		appliedResults = append(appliedResults, applied...)
		skippedResults = append(skippedResults, skipped...)
		appliedCount += len(applied)

		// If a non-stackable promotion was applied, stop processing further groups
		if s.hasNonStackable(applied, candidates) {
			break
		}
	}

	return s.buildPromotionSummary(cart, effectivePrices, appliedResults, skippedResults), nil
}

// initEffectivePrices builds the initial price map from cart items (O(n) where n = items)
func (s *PromotionServiceImpl) initEffectivePrices(
	cart *model.CartValidationRequest,
) map[string]int64 {
	prices := make(map[string]int64, len(cart.Items))
	for _, item := range cart.Items {
		prices[item.ItemID] = item.PriceCents
	}
	return prices
}

// filterValidPromotions validates general conditions and splits into valid vs skipped (O(p) where p = promotions)
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

// groupByPriority groups pre-sorted promotions by priority in a single pass (O(p))
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

// evaluateCandidates calculates discounts for a priority group and returns sorted candidates (O(g × n))
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

// applyBestCandidates applies candidates respecting stacking rules and updates effective prices (O(c × n))
func (s *PromotionServiceImpl) applyBestCandidates(
	candidates []promotionCandidate,
	effectivePrices map[string]int64,
	alreadyAppliedCount int,
) (applied []model.PromotionValidationResult, skipped []model.PromotionValidationResult) {
	for _, c := range candidates {
		canStack := c.promotion.CanStackWithOtherPromotions != nil &&
			*c.promotion.CanStackWithOtherPromotions

		if (alreadyAppliedCount+len(applied)) > 0 && !canStack {
			skipped = append(
				skipped,
				skippedResult(c.promotion, "Cannot stack with other promotions"),
			)
			continue
		}

		// Update effective prices with this promotion's discounts
		for _, d := range c.result.ItemDiscounts {
			if d.FinalCents >= 0 {
				effectivePrices[d.ItemID] = d.FinalCents
			}
		}

		applied = append(applied, *c.result)

		if !canStack {
			break
		}
	}

	return applied, skipped
}

// hasNonStackable checks if any applied candidate was non-stackable (signals to stop further groups)
func (s *PromotionServiceImpl) hasNonStackable(
	applied []model.PromotionValidationResult,
	candidates []promotionCandidate,
) bool {
	for _, c := range candidates {
		for _, a := range applied {
			if a.Promotion != nil && a.Promotion.ID == c.promotion.ID {
				canStack := c.promotion.CanStackWithOtherPromotions != nil &&
					*c.promotion.CanStackWithOtherPromotions
				if !canStack {
					return true
				}
			}
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
