package promotionStrategy

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"

	"ecommerce-be/common/helper"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
)

// BuyXGetYStrategy implements PromotionStrategy for buy_x_get_y promotion type.
//
// Business Logic:
//   The customer buys buy_quantity units and gets get_quantity units free. Complete sets
//   are computed as total_eligible_qty / (buy_quantity + get_quantity). An optional
//   max_sets cap limits how many sets can apply per order.
//
//   Two modes are supported:
//
//   1. Same-reward (is_same_reward=true, default):
//      Reward items come from the same pool as qualifying items. Items are grouped by
//      scope_type (same_variant | same_product | same_category). Within each group the
//      highest-priced units are reserved as "paid" and the cheapest remaining units
//      become free. This ensures the customer always pays the higher prices.
//
//   2. Cross-product reward (is_same_reward=false):
//      Buy items come from the promotion scope, reward items come from get_product_id.
//      The reward product must already be in the cart; the strategy does not auto-add it.
//      Cheapest reward units are made free first.
//
// Config Fields:
//   - buy_quantity   (required) : units the customer must buy
//   - get_quantity   (required) : units that become free
//   - max_sets       (optional) : cap on complete sets per order
//   - is_same_reward (optional, default true)
//   - scope_type     (required when is_same_reward=true) : "same_variant" | "same_product" | "same_category"
//   - get_product_id (required when is_same_reward=false) : specific reward product
//
// Example (same-reward, scope_type=same_product, buy 2 get 1):
//   Config: { buy_quantity: 2, get_quantity: 1, is_same_reward: true, scope_type: "same_product" }
//   Cart:
//     Item A  product 1, variant X   $1000 x1
//     Item B  product 1, variant Y   $800  x1
//     Item C  product 1, variant Z   $600  x1
//   All three belong to the same product-1 group  =>  totalQty = 3
//   Complete sets = 3 / (2+1) = 1
//   Paid  (2 highest): Item A ($1000) + Item B ($800) = $1800  (customer pays these)
//   Free  (1 cheapest): Item C ($600) = $600           (discount)
//   Total discount = 60000,  Final subtotal = 240000 - 60000 = 180000
type BuyXGetYStrategy struct{}

type bxgyGroupLine struct {
	item           model.CartItem
	effectivePrice int64
	quantity       int
}

type bxgyGroup struct {
	key      string
	lines    []bxgyGroupLine
	totalQty int
}

type bxgyRewardLine struct {
	item           model.CartItem
	effectivePrice int64
	quantity       int
}

// NewBuyXGetYStrategy creates a new BuyXGetYStrategy
func NewBuyXGetYStrategy() PromotionStrategy {
	return &BuyXGetYStrategy{}
}

// DescribeConfig returns the supported Buy X Get Y fields and setup guidance.
func (s *BuyXGetYStrategy) DescribeConfig() model.PromotionStrategyDescriptor {
	return model.PromotionStrategyDescriptor{
		PromotionType: entity.PromoTypeBuyXGetY,
		Name:          "Buy X Get Y",
		Description:   "Makes the configured number of reward items free after the required buy quantity is satisfied.",
		Fields: []model.PromotionConfigFieldDescriptor{
			{
				Name:        "buy_quantity",
				Type:        "int",
				Required:    true,
				Description: "Number of qualifying units the customer must buy.",
			},
			{
				Name:        "get_quantity",
				Type:        "int",
				Required:    true,
				Description: "Number of reward units that become free.",
			},
			{
				Name:        "max_sets",
				Type:        "int",
				Required:    false,
				Description: "Optional cap on how many complete buy/get sets apply per order.",
			},
			{
				Name:         "is_same_reward",
				Type:         "bool",
				Required:     false,
				Description:  "Defaults to true; when true the reward comes from the same eligible pool.",
				DefaultValue: true,
			},
			{
				Name:          "scope_type",
				Type:          "string",
				Required:      false,
				Description:   "Required when is_same_reward=true; controls same-pool grouping.",
				AllowedValues: []string{"same_variant", "same_product", "same_category"},
			},
			{
				Name:        "get_product_id",
				Type:        "uint",
				Required:    false,
				Description: "Required when is_same_reward=false; specific reward product that must already be present in the cart.",
			},
		},
		BestPractices: []string{
			"Default to same_product unless a broader grouped reward is a deliberate business decision.",
			"Do not enable stacking unless double-discounting behavior is explicitly acceptable for the catalog.",
			"Cross-product rewards should point at a low-risk accessory or add-on rather than a high-value item.",
		},
	}
}

// ValidateConfig enforces the two supported config shapes:
// 1. same-reward mode => scope_type required, get_product_id forbidden
// 2. cross-product reward mode => get_product_id required, scope_type forbidden
func (s *BuyXGetYStrategy) ValidateConfig(config map[string]interface{}) error {
	buyXGetYConfig, err := parseBuyXGetYConfig(config)
	if err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid buy_x_get_y config structure",
		)
	}

	if buyXGetYConfig.BuyQuantity <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"buy_quantity must be greater than 0",
		)
	}

	if buyXGetYConfig.GetQuantity <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"get_quantity must be greater than 0",
		)
	}

	if buyXGetYConfig.MaxSets != nil && *buyXGetYConfig.MaxSets <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"max_sets must be greater than 0 if specified",
		)
	}

	if isSameRewardMode(buyXGetYConfig) {
		return validateSameRewardConfig(buyXGetYConfig)
	}

	return validateCrossRewardConfig(buyXGetYConfig)
}

func validateSameRewardConfig(config model.BuyXGetYConfig) error {
	if config.GetProductID != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"get_product_id must be omitted when is_same_reward is true",
		)
	}

	switch config.ScopeType {
	case model.BuyXGetYScopeSameVariant,
		model.BuyXGetYScopeSameProduct,
		model.BuyXGetYScopeSameCategory:
		return nil
	default:
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"scope_type must be one of same_variant, same_product, or same_category when is_same_reward is true",
		)
	}
}

func validateCrossRewardConfig(config model.BuyXGetYConfig) error {
	if config.GetProductID == nil || *config.GetProductID == 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"get_product_id is required when is_same_reward is false",
		)
	}

	if config.ScopeType != "" {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"scope_type must be omitted when is_same_reward is false",
		)
	}

	return nil
}

func isSameRewardMode(config model.BuyXGetYConfig) bool {
	return config.IsSameReward != nil && *config.IsSameReward
}

// CalculateDiscount updates the passed AppliedPromotionSummary in-place and
// returns an error if the promotion cannot be applied.
//
// Instead of duplicating the complex grouping/allocation logic, this method derives
// an effectivePrices map and eligible []CartItem from the summary, then delegates to
// the existing helpers (calculateSameRewardDiscount / calculateCrossRewardDiscount).
func (s *BuyXGetYStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	summary *model.AppliedPromotionSummary,
	eligibleItems []string,
) error {
	config, err := parseBuyXGetYConfig(promotion.DiscountConfig)
	if err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid buy_x_get_y promotion configuration",
		)
	}

	// Derive per-unit effective prices from summary for ALL items (buy + reward).
	effectivePrices := make(map[string]int64, len(summary.Items))
	for _, item := range summary.Items {
		if item.Quantity > 0 {
			effectivePrices[item.ItemID] = item.FinalPriceCents / int64(item.Quantity)
		}
	}

	// Derive eligible CartItems from the passed eligibleItems set.
	eligibleItemsSet := helper.ToSet(eligibleItems)
	eligibleCartItems := make([]model.CartItem, 0, len(eligibleItems))
	for _, item := range cart.Items {
		if eligibleItemsSet[item.ItemID] {
			eligibleCartItems = append(eligibleCartItems, item)
		}
	}
	if len(eligibleCartItems) == 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Cart items do not match promotion scope",
		)
	}

	// Dispatch to existing V1 logic which returns a fully computed result.
	var v1Result *model.PromotionValidationResult
	if isSameRewardMode(config) {
		v1Result = s.calculateSameRewardDiscount(
			promotion,
			cart,
			eligibleCartItems,
			effectivePrices,
			config,
		)
	} else {
		v1Result = s.calculateCrossRewardDiscount(promotion, cart, eligibleCartItems, effectivePrices, config)
	}

	if !v1Result.IsValid {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(v1Result.Reason)
	}

	// Convert V1 ItemDiscounts into ItemDiscountDetail for the shared summary updater.
	itemDiscounts := make([]ItemDiscountDetail, 0, len(v1Result.ItemDiscounts))
	for _, d := range v1Result.ItemDiscounts {
		itemDiscounts = append(itemDiscounts, ItemDiscountDetail{
			ItemID:        d.ItemID,
			DiscountCents: d.DiscountCents,
			FreeQuantity:  d.FreeQuantity,
		})
	}

	ApplyDiscountToSummary(summary, promotion, itemDiscounts, v1Result.DiscountCents, 0)
	return nil
}

// calculateSameRewardDiscount handles:
// - same_variant
// - same_product
// - same_category
//
// It groups eligible cart units, computes complete sets in each group, uses the highest-priced
// units to satisfy the "buy" side, and gives the cheapest remaining units for free.
func (s *BuyXGetYStrategy) calculateSameRewardDiscount(
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	eligibleItems []model.CartItem,
	effectivePrices map[string]int64,
	config model.BuyXGetYConfig,
) *model.PromotionValidationResult {
	result := &model.PromotionValidationResult{IsValid: false}
	groups := s.groupEligibleItems(eligibleItems, effectivePrices, config.ScopeType)

	itemDiscountTotals := make(map[string]int64)
	itemFreeQuantities := make(map[string]int)
	var totalDiscountCents int64

	for _, group := range groups {
		totalItemsPerSet := config.BuyQuantity + config.GetQuantity
		if group.totalQty < totalItemsPerSet {
			continue
		}

		completeSets := group.totalQty / totalItemsPerSet
		if config.MaxSets != nil && completeSets > *config.MaxSets {
			completeSets = *config.MaxSets
		}
		if completeSets == 0 {
			continue
		}

		paidCount := completeSets * config.BuyQuantity
		freeCount := completeSets * config.GetQuantity
		groupDiscount := allocateSameRewardGroupDiscounts(group.lines, paidCount, freeCount)
		mergeItemDiscountMaps(
			itemDiscountTotals,
			itemFreeQuantities,
			&totalDiscountCents,
			groupDiscount,
		)
	}

	if totalDiscountCents == 0 {
		result.Reason = "Not enough items to qualify for buy X get Y promotion"
		return result
	}

	result.IsValid = true
	result.DiscountCents = totalDiscountCents
	result.ItemDiscounts = buildBuyXGetYItemDiscounts(
		promotion,
		cart.Items,
		effectivePrices,
		itemDiscountTotals,
		itemFreeQuantities,
	)
	return result
}

// calculateCrossRewardDiscount handles the "buy this, get that product free" case.
// The reward product must already exist in the cart; the strategy does not auto-add it.
// Buy items come from the promotion scope, while reward items come from get_product_id.
func (s *BuyXGetYStrategy) calculateCrossRewardDiscount(
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	eligibleItems []model.CartItem,
	effectivePrices map[string]int64,
	config model.BuyXGetYConfig,
) *model.PromotionValidationResult {
	result := &model.PromotionValidationResult{IsValid: false}

	buyUnitCount := countCrossRewardBuyUnits(eligibleItems, effectivePrices, *config.GetProductID)
	rewardLines, rewardUnitCount := collectCrossRewardRewardLines(
		cart.Items,
		effectivePrices,
		*config.GetProductID,
	)

	completeSets, reason := calculateCrossRewardCompleteSets(
		buyUnitCount,
		rewardUnitCount,
		config,
	)
	if completeSets == 0 {
		result.Reason = reason
		return result
	}

	sortCrossRewardLinesByPriceAsc(rewardLines)

	freeCount := completeSets * config.GetQuantity
	itemDiscountTotals, itemFreeQuantities, totalDiscountCents := applyFreeRewardLines(
		rewardLines,
		freeCount,
	)

	result.IsValid = true
	result.DiscountCents = totalDiscountCents
	result.ItemDiscounts = buildBuyXGetYItemDiscounts(
		promotion,
		cart.Items,
		effectivePrices,
		itemDiscountTotals,
		itemFreeQuantities,
	)
	return result
}

func countCrossRewardBuyUnits(
	eligibleItems []model.CartItem,
	effectivePrices map[string]int64,
	rewardProductID uint,
) int {
	total := 0
	for _, item := range eligibleItems {
		if item.ProductID == rewardProductID {
			continue
		}
		effectivePrice := effectivePrices[item.ItemID]
		if effectivePrice <= 0 {
			continue
		}
		total += item.Quantity
	}
	return total
}

func collectCrossRewardRewardLines(
	cartItems []model.CartItem,
	effectivePrices map[string]int64,
	rewardProductID uint,
) ([]bxgyRewardLine, int) {
	rewardLines := make([]bxgyRewardLine, 0)
	totalUnits := 0
	for _, item := range cartItems {
		if item.ProductID != rewardProductID {
			continue
		}
		effectivePrice := effectivePrices[item.ItemID]
		if effectivePrice <= 0 {
			continue
		}
		rewardLines = append(rewardLines, bxgyRewardLine{
			item:           item,
			effectivePrice: effectivePrice,
			quantity:       item.Quantity,
		})
		totalUnits += item.Quantity
	}
	return rewardLines, totalUnits
}

func calculateCrossRewardCompleteSets(
	buyUnitCount int,
	rewardUnitCount int,
	config model.BuyXGetYConfig,
) (int, string) {
	if buyUnitCount < config.BuyQuantity {
		return 0, "Not enough qualifying buy items for buy X get Y promotion"
	}
	if rewardUnitCount < config.GetQuantity {
		return 0, "Reward item must be present in cart for buy X get Y promotion"
	}

	completeSets := buyUnitCount / config.BuyQuantity
	rewardSets := rewardUnitCount / config.GetQuantity
	if rewardSets < completeSets {
		completeSets = rewardSets
	}
	if config.MaxSets != nil && completeSets > *config.MaxSets {
		completeSets = *config.MaxSets
	}
	if completeSets == 0 {
		return 0, "Not enough items to qualify for buy X get Y promotion"
	}

	return completeSets, ""
}

func sortCrossRewardLinesByPriceAsc(rewardLines []bxgyRewardLine) {
	sort.SliceStable(rewardLines, func(i, j int) bool {
		if rewardLines[i].effectivePrice == rewardLines[j].effectivePrice {
			return rewardLines[i].item.ItemID < rewardLines[j].item.ItemID
		}
		return rewardLines[i].effectivePrice < rewardLines[j].effectivePrice
	})
}

func applyFreeRewardLines(
	rewardLines []bxgyRewardLine,
	freeCount int,
) (map[string]int64, map[string]int, int64) {
	itemDiscountTotals := make(map[string]int64)
	itemFreeQuantities := make(map[string]int)
	var totalDiscountCents int64

	remaining := freeCount
	for _, line := range rewardLines {
		if remaining == 0 {
			break
		}

		freeQty := line.quantity
		if freeQty > remaining {
			freeQty = remaining
		}
		if freeQty <= 0 {
			continue
		}

		itemDiscountTotals[line.item.ItemID] += int64(freeQty) * line.effectivePrice
		itemFreeQuantities[line.item.ItemID] += freeQty
		totalDiscountCents += int64(freeQty) * line.effectivePrice
		remaining -= freeQty
	}
	return itemDiscountTotals, itemFreeQuantities, totalDiscountCents
}

// groupEligibleItems builds quantity-aware groups for same-reward mode.
// This avoids per-unit expansion for better performance on high-quantity carts.
func (s *BuyXGetYStrategy) groupEligibleItems(
	items []model.CartItem,
	effectivePrices map[string]int64,
	scopeType model.BuyXGetYScopeType,
) []bxgyGroup {
	groupMap := make(map[string]*bxgyGroup)

	for _, item := range items {
		effectivePrice := effectivePrices[item.ItemID]
		if effectivePrice <= 0 {
			continue
		}

		key := buyXGetYGroupKey(item, scopeType)
		if key == "" {
			continue
		}

		group := groupMap[key]
		if group == nil {
			group = &bxgyGroup{key: key}
			groupMap[key] = group
		}
		group.lines = append(group.lines, bxgyGroupLine{
			item:           item,
			effectivePrice: effectivePrice,
			quantity:       item.Quantity,
		})
		group.totalQty += item.Quantity
	}

	keys := make([]string, 0, len(groupMap))
	for key := range groupMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	groups := make([]bxgyGroup, 0, len(keys))
	for _, key := range keys {
		groups = append(groups, *groupMap[key])
	}
	return groups
}

type sameRewardGroupDiscount struct {
	itemDiscountTotals map[string]int64
	itemFreeQuantities map[string]int
	totalDiscountCents int64
}

// allocateSameRewardGroupDiscounts applies pricing rules without expanding to unit slices:
// 1) reserve paid quantities from highest price to lowest price
// 2) discount free quantities from the cheapest remaining quantities
func allocateSameRewardGroupDiscounts(
	lines []bxgyGroupLine,
	paidCount int,
	freeCount int,
) sameRewardGroupDiscount {
	result := sameRewardGroupDiscount{
		itemDiscountTotals: make(map[string]int64),
		itemFreeQuantities: make(map[string]int),
	}
	if freeCount <= 0 || len(lines) == 0 {
		return result
	}

	byPriceDesc := append([]bxgyGroupLine(nil), lines...)
	sort.SliceStable(byPriceDesc, func(i, j int) bool {
		if byPriceDesc[i].effectivePrice == byPriceDesc[j].effectivePrice {
			return byPriceDesc[i].item.ItemID < byPriceDesc[j].item.ItemID
		}
		return byPriceDesc[i].effectivePrice > byPriceDesc[j].effectivePrice
	})

	remainingQtyByItemID := make(map[string]int, len(lines))
	for _, line := range byPriceDesc {
		remainingQtyByItemID[line.item.ItemID] += line.quantity
	}

	paidRemaining := paidCount
	for _, line := range byPriceDesc {
		if paidRemaining == 0 {
			break
		}
		available := remainingQtyByItemID[line.item.ItemID]
		if available <= 0 {
			continue
		}
		consume := available
		if consume > paidRemaining {
			consume = paidRemaining
		}
		remainingQtyByItemID[line.item.ItemID] -= consume
		paidRemaining -= consume
	}

	byPriceAsc := append([]bxgyGroupLine(nil), lines...)
	sort.SliceStable(byPriceAsc, func(i, j int) bool {
		if byPriceAsc[i].effectivePrice == byPriceAsc[j].effectivePrice {
			return byPriceAsc[i].item.ItemID < byPriceAsc[j].item.ItemID
		}
		return byPriceAsc[i].effectivePrice < byPriceAsc[j].effectivePrice
	})

	freeRemaining := freeCount
	for _, line := range byPriceAsc {
		if freeRemaining == 0 {
			break
		}
		available := remainingQtyByItemID[line.item.ItemID]
		if available <= 0 {
			continue
		}
		freeQty := available
		if freeQty > freeRemaining {
			freeQty = freeRemaining
		}
		result.itemDiscountTotals[line.item.ItemID] += int64(freeQty) * line.effectivePrice
		result.itemFreeQuantities[line.item.ItemID] += freeQty
		result.totalDiscountCents += int64(freeQty) * line.effectivePrice
		freeRemaining -= freeQty
	}

	return result
}

func mergeItemDiscountMaps(
	itemDiscountTotals map[string]int64,
	itemFreeQuantities map[string]int,
	totalDiscountCents *int64,
	group sameRewardGroupDiscount,
) {
	for itemID, cents := range group.itemDiscountTotals {
		itemDiscountTotals[itemID] += cents
	}
	for itemID, qty := range group.itemFreeQuantities {
		itemFreeQuantities[itemID] += qty
	}
	*totalDiscountCents += group.totalDiscountCents
}

// buildBuyXGetYItemDiscounts converts the aggregated per-line discount totals into the
// ItemDiscount format used by the promotion engine summary and stacking logic.
func buildBuyXGetYItemDiscounts(
	promotion *entity.Promotion,
	cartItems []model.CartItem,
	effectivePrices map[string]int64,
	itemDiscountTotals map[string]int64,
	itemFreeQuantities map[string]int,
) []model.ItemDiscount {
	itemDiscounts := make([]model.ItemDiscount, 0, len(itemDiscountTotals))
	for _, item := range cartItems {
		itemDiscount := itemDiscountTotals[item.ItemID]
		if itemDiscount <= 0 {
			continue
		}

		effectivePrice := effectivePrices[item.ItemID]
		finalCents := effectivePrice - (itemDiscount / int64(item.Quantity))
		if finalCents < 0 {
			finalCents = 0
		}

		itemDiscounts = append(itemDiscounts, model.ItemDiscount{
			ItemID:        item.ItemID,
			ProductID:     item.ProductID,
			PromotionID:   promotion.ID,
			PromotionName: promotion.Name,
			DiscountCents: itemDiscount,
			OriginalCents: effectivePrice,
			FinalCents:    finalCents,
			FreeQuantity:  itemFreeQuantities[item.ItemID],
		})
	}
	return itemDiscounts
}

// buyXGetYGroupKey chooses the grouping key for same-reward mode.
// All units with the same key compete inside the same Buy X Get Y pool.
func buyXGetYGroupKey(item model.CartItem, scopeType model.BuyXGetYScopeType) string {
	switch scopeType {
	case model.BuyXGetYScopeSameVariant:
		if item.VariantID == nil {
			return ""
		}
		return "variant:" + strconv.FormatUint(uint64(*item.VariantID), 10)
	case model.BuyXGetYScopeSameProduct:
		return "product:" + strconv.FormatUint(uint64(item.ProductID), 10)
	case model.BuyXGetYScopeSameCategory:
		return "category:" + strconv.FormatUint(uint64(item.CategoryID), 10)
	default:
		return ""
	}
}

// parseBuyXGetYConfig normalizes defaults before unmarshalling:
// - is_same_reward defaults to true
// - scope_type defaults to same_product when same-reward mode is used
func parseBuyXGetYConfig(config map[string]interface{}) (model.BuyXGetYConfig, error) {
	normalized := make(map[string]interface{}, len(config)+2)
	for key, value := range config {
		normalized[key] = value
	}

	configJSON, err := json.Marshal(normalized)
	if err != nil {
		return model.BuyXGetYConfig{}, err
	}

	var buyXGetYConfig model.BuyXGetYConfig
	if err := json.Unmarshal(configJSON, &buyXGetYConfig); err != nil {
		return model.BuyXGetYConfig{}, err
	}

	if buyXGetYConfig.IsSameReward == nil {
		buyXGetYConfig.IsSameReward = helper.BoolPtr(true)
	}

	if *buyXGetYConfig.IsSameReward && buyXGetYConfig.ScopeType == "" {
		buyXGetYConfig.ScopeType = model.BuyXGetYScopeSameProduct
	}

	return buyXGetYConfig, nil
}
