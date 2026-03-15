package promotionStrategy

import (
	"context"
	"encoding/json"

	"ecommerce-be/common/helper"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/repository"
)

// FixedAmountStrategy implements PromotionStrategy for fixed_amount promotion type
type FixedAmountStrategy struct{}

// NewFixedAmountStrategy creates a new FixedAmountStrategy
func NewFixedAmountStrategy() PromotionStrategy {
	return &FixedAmountStrategy{}
}

// DescribeConfig returns the supported fixed-amount discount fields and setup guidance.
func (s *FixedAmountStrategy) DescribeConfig() model.PromotionStrategyDescriptor {
	return model.PromotionStrategyDescriptor{
		PromotionType: entity.PromoTypeFixedAmount,
		Name:          "Fixed Amount Discount",
		Description:   "Applies a fixed cart-level discount distributed across eligible items.",
		Fields: []model.PromotionConfigFieldDescriptor{
			{
				Name:        "amount_cents",
				Type:        "int64",
				Required:    true,
				Description: "Fixed discount amount in smallest currency unit.",
			},
			{
				Name:        "min_order_cents",
				Type:        "int64",
				Required:    false,
				Description: "Optional minimum eligible cart total (in smallest currency unit) required to apply the discount.",
			},
		},
		BestPractices: []string{
			"Use a minimum order amount to protect margin on low-value orders.",
			"Keep the amount below the typical cart subtotal to avoid zeroing most carts.",
			"Disable stacking unless combined discounts are intentionally allowed.",
		},
	}
}

// ValidateConfig validates the fixed amount configuration
func (s *FixedAmountStrategy) ValidateConfig(config map[string]interface{}) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("Invalid config format")
	}

	var fixedConfig model.FixedAmountConfig
	if err := json.Unmarshal(configJSON, &fixedConfig); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid fixed_amount config structure",
		)
	}

	if fixedConfig.AmountCents <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"amount_cents must be greater than 0",
		)
	}

	if fixedConfig.MinOrderCents != nil && *fixedConfig.MinOrderCents < 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"min_order_cents must be greater than or equal to 0",
		)
	}

	return nil
}

// CalculateDiscount calculates per-item fixed amount discount (distributed proportionally)
func (s *FixedAmountStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	effectivePrices map[string]int64,
) (*model.PromotionValidationResult, error) {
	result := &model.PromotionValidationResult{
		IsValid: false,
	}

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.FixedAmountConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		result.Reason = "Invalid promotion configuration"
		return result, nil
	}

	eligibleItems, err := s.getEligibleItems(ctx, promotion, cart)
	if err != nil {
		return nil, err
	}
	if len(eligibleItems) == 0 {
		result.Reason = "Cart items do not match promotion scope"
		return result, nil
	}

	// Calculate total effective value over eligible items only.
	var totalEffective int64
	for _, item := range eligibleItems {
		totalEffective += effectivePrices[item.ItemID] * int64(item.Quantity)
	}

	if totalEffective <= 0 {
		result.Reason = "No eligible items"
		return result, nil
	}

	if config.MinOrderCents != nil && totalEffective < *config.MinOrderCents {
		result.Reason = "Minimum order amount not met"
		return result, nil
	}

	discountCents := config.AmountCents
	if discountCents > totalEffective {
		discountCents = totalEffective
	}
	if promotion.MaxDiscountAmountCents != nil &&
		discountCents > *promotion.MaxDiscountAmountCents {
		discountCents = *promotion.MaxDiscountAmountCents
	}

	// Distribute discount proportionally across eligible items only.
	var itemDiscounts []model.ItemDiscount
	var distributed int64

	for i, item := range eligibleItems {
		effectivePrice := effectivePrices[item.ItemID]
		if effectivePrice <= 0 {
			continue
		}

		itemTotal := effectivePrice * int64(item.Quantity)
		var itemDiscount int64

		if i == len(cart.Items)-1 {
			// Last item gets remainder to avoid rounding issues
			itemDiscount = discountCents - distributed
		} else {
			itemDiscount = discountCents * itemTotal / totalEffective
		}

		if itemDiscount > itemTotal {
			itemDiscount = itemTotal
		}
		if itemDiscount > 0 {
			distributed += itemDiscount
			itemDiscounts = append(itemDiscounts, model.ItemDiscount{
				ItemID:        item.ItemID,
				ProductID:     item.ProductID,
				PromotionID:   promotion.ID,
				PromotionName: promotion.Name,
				DiscountCents: itemDiscount,
				OriginalCents: effectivePrice,
				FinalCents:    effectivePrice - (itemDiscount / int64(item.Quantity)),
			})
		}
	}

	result.IsValid = true
	result.DiscountCents = discountCents
	result.ItemDiscounts = itemDiscounts
	return result, nil
}

// CalculateDiscountV2 is the enhanced version of CalculateDiscount that will update the summary in-place and
// return error if promotion cannot be applied
func (s *FixedAmountStrategy) CalculateDiscountV2(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	summary *model.AppliedPromotionSummary,
	eligibleItems []string,
) error {
	eligibleItemsSet := helper.ToSet(eligibleItems)

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.FixedAmountConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid fixed_amount promotion configuration",
		)
	}

	// Collect eligible summary items and compute total effective value using current
	// FinalPriceCents (line total) so previously applied promotions are factored in.
	var eligibleSummaryItems []*model.CartItemSummary
	var totalEffective int64
	for i := range summary.Items {
		item := &summary.Items[i]
		if !eligibleItemsSet[item.ItemID] {
			continue
		}
		eligibleSummaryItems = append(eligibleSummaryItems, item)
		totalEffective += item.FinalPriceCents // FinalPriceCents is the line total (PriceCents * Quantity)
	}

	if totalEffective <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"No eligible items for fixed amount promotion",
		)
	}

	if config.MinOrderCents != nil && totalEffective < *config.MinOrderCents {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("Minimum order amount not met")
	}

	discountCents := config.AmountCents
	if discountCents > totalEffective {
		discountCents = totalEffective
	}

	// Distribute discount proportionally across eligible items; last item gets the
	// remainder to prevent cents from being lost to integer truncation.
	itemDiscounts := make([]ItemDiscountDetail, 0, len(eligibleSummaryItems))
	var distributed int64

	for i, item := range eligibleSummaryItems {
		if item.FinalPriceCents <= 0 {
			continue
		}

		// FinalPriceCents is the line total; use it directly for proportional distribution.
		var itemDiscount int64
		if i == len(eligibleSummaryItems)-1 {
			itemDiscount = discountCents - distributed
		} else {
			itemDiscount = discountCents * item.FinalPriceCents / totalEffective
		}

		if itemDiscount > item.FinalPriceCents {
			itemDiscount = item.FinalPriceCents
		}
		if itemDiscount > 0 {
			distributed += itemDiscount
			itemDiscounts = append(itemDiscounts, ItemDiscountDetail{
				ItemID:        item.ItemID,
				DiscountCents: itemDiscount,
			})
		}
	}

	if distributed == 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"No discount applicable for fixed amount promotion",
		)
	}

	ApplyDiscountToSummary(summary, promotion, itemDiscounts, distributed, 0)
	return nil
}

// getEligibleItems returns the cart lines that are eligible for this promotion
// based on its AppliesTo scope. Only these items participate in the fixed
// amount discount calculation.
func (s *FixedAmountStrategy) getEligibleItems(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
) ([]model.CartItem, error) {
	switch promotion.AppliesTo {
	case entity.ScopeAllProducts:
		// All cart items are eligible; return a shallow copy for safety.
		return append([]model.CartItem(nil), cart.Items...), nil
	case entity.ScopeSpecificProducts:
		return s.filterProductScopedItems(ctx, promotion.ID, cart)
	case entity.ScopeSpecificCategories:
		return s.filterCategoryScopedItems(ctx, promotion.ID, cart)
	case entity.ScopeSpecificCollections:
		// Collection-scoped fixed-amount is not wired yet; keep behavior explicit.
		return nil, nil
	default:
		return nil, nil
	}
}

// filterProductScopedItems returns only the cart lines whose product IDs are
// linked to the promotion through the product scope table.
func (s *FixedAmountStrategy) filterProductScopedItems(
	ctx context.Context,
	promotionID uint,
	cart *model.CartValidationRequest,
) ([]model.CartItem, error) {
	productIDs := make([]uint, len(cart.Items))
	for i, item := range cart.Items {
		productIDs[i] = item.ProductID
	}

	repo := repository.NewPromotionProductScopeRepository()
	linkedProducts, _, err := repo.GetPromotionProducts(
		ctx,
		promotionID,
		productIDs,
		0,
		len(productIDs),
	)
	if err != nil {
		return nil, err
	}

	allowedProducts := make(map[uint]struct{}, len(linkedProducts))
	for _, product := range linkedProducts {
		allowedProducts[product.ProductID] = struct{}{}
	}

	filtered := make([]model.CartItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		if _, ok := allowedProducts[item.ProductID]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

// filterCategoryScopedItems returns only the cart lines whose category IDs are
// linked to the promotion through the category scope table.
func (s *FixedAmountStrategy) filterCategoryScopedItems(
	ctx context.Context,
	promotionID uint,
	cart *model.CartValidationRequest,
) ([]model.CartItem, error) {
	categoryIDs := make([]uint, len(cart.Items))
	for i, item := range cart.Items {
		categoryIDs[i] = item.CategoryID
	}

	repo := repository.NewPromotionCategoryScopeRepository()
	linkedCategories, _, err := repo.GetPromotionCategories(
		ctx,
		promotionID,
		categoryIDs,
		0,
		len(categoryIDs),
	)
	if err != nil {
		return nil, err
	}

	allowedCategories := make(map[uint]struct{}, len(linkedCategories))
	for _, category := range linkedCategories {
		allowedCategories[category.CategoryID] = struct{}{}
	}

	filtered := make([]model.CartItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		if _, ok := allowedCategories[item.CategoryID]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}
