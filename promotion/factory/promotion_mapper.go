package factory

import (
	"encoding/json"
	"time"

	"ecommerce-be/common/db"
	commonModel "ecommerce-be/common/model"
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
)

// configMoneyKeyMap maps major-unit config keys (client-facing) to the internal
// cents keys the promotion strategies read from the discount_config JSONB.
var configMoneyKeyMap = map[string]string{
	"amount":                "amount_cents",
	"min_order":             "min_order_cents",
	"max_discount":          "max_discount_cents",
	"bundle_price":          "bundle_price_cents",
	"max_shipping_discount": "max_shipping_discount_cents",
	"min_purchase_amount":   "min_purchase_amount_cents",
	"max_discount_amount":   "max_discount_amount_cents",
}

// ConvertConfigMoneyToCents converts major-unit money values in a discount
// config map to cents keys (e.g. "amount": 50.00 → "amount_cents": 5000).
// Non-money keys are left untouched; percentages stay plain numbers.
func ConvertConfigMoneyToCents(config map[string]any, ccy commonModel.CurrencyInfo) (map[string]any, error) {
	out := make(map[string]any, len(config))
	for k, v := range config {
		if centsKey, ok := configMoneyKeyMap[k]; ok {
			major, ok := v.(float64)
			if !ok {
				out[k] = v
				continue
			}
			cents, err := ccy.ToCents(major)
			if err != nil {
				return nil, err
			}
			out[centsKey] = cents
			continue
		}
		out[k] = v
	}
	return out, nil
}

// ConvertConfigCentsToMoney converts cents keys in a discount config map back to
// Money objects under their major-unit key names for responses.
func ConvertConfigCentsToMoney(config map[string]any, ccy commonModel.CurrencyInfo) map[string]any {
	out := make(map[string]any, len(config))
	for k, v := range config {
		switch k {
		case "amount_cents", "min_order_cents", "max_discount_cents", "bundle_price_cents", "max_shipping_discount_cents", "min_purchase_amount_cents", "max_discount_amount_cents":
			cents, ok := toInt64(v)
			if ok {
				majorKey := centsToMajorKey(k)
				out[majorKey] = commonModel.NewMoney(cents, ccy)
				continue
			}
		}
		out[k] = v
	}
	return out
}

// toInt64 coerces a JSON-decoded number (float64 or int64) to int64.
func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	}
	return 0, false
}

func centsToMajorKey(centsKey string) string {
	for major, cents := range configMoneyKeyMap {
		if cents == centsKey {
			return major
		}
	}
	return centsKey
}

// PromotionRequestToEntity converts CreatePromotionRequest to Promotion entity
func PromotionRequestToEntity(req model.CreatePromotionRequest, sellerID uint) *entity.Promotion {
	promotion := &entity.Promotion{
		SellerID:                    sellerID,
		Name:                        req.Name,
		DisplayName:                 req.DisplayName,
		Slug:                        req.Slug,
		Description:                 req.Description,
		PromotionType:               req.PromotionType,
		DiscountConfig:              db.JSONMap(req.DiscountConfig),
		AppliesTo:                   req.AppliesTo,
		EligibleFor:                 req.EligibleFor,
		CustomerSegmentID:           req.CustomerSegmentID,
		UsageLimitTotal:             req.UsageLimitTotal,
		UsageLimitPerCustomer:       req.UsageLimitPerCustomer,
		AutoStart:                   req.AutoStart,
		AutoEnd:                     req.AutoEnd,
		CanStackWithOtherPromotions: req.CanStackWithOtherPromotions,
		CanStackWithCoupons:         req.CanStackWithCoupons,
		ShowOnStorefront:            req.ShowOnStorefront,
		BadgeText:                   req.BadgeText,
		BadgeColor:                  req.BadgeColor,
		SaleID:                      req.SaleID,
	}

	// Parse StartsAt
	if req.StartsAt != nil {
		if startsAt, err := time.Parse(time.RFC3339, *req.StartsAt); err == nil {
			promotion.StartsAt = &startsAt
		}
	}

	// Parse EndsAt
	if req.EndsAt != nil {
		if endsAt, err := time.Parse(time.RFC3339, *req.EndsAt); err == nil {
			promotion.EndsAt = &endsAt
		}
	}

	// Set Status (default to draft if not provided)
	if req.Status != "" {
		promotion.Status = req.Status
	} else {
		promotion.Status = entity.StatusDraft
	}

	// Set Priority (default to 0 if not provided)
	if req.Priority != nil {
		promotion.Priority = *req.Priority
	} else {
		promotion.Priority = 0
	}

	return promotion
}

// PromotionEntityToResponse converts Promotion entity to PromotionResponse
func PromotionEntityToResponse(
	promotion *entity.Promotion,
	ccy commonModel.CurrencyInfo,
) *model.PromotionResponse {
	convertedConfig := ConvertConfigCentsToMoney(map[string]any(promotion.DiscountConfig), ccy)
	response := &model.PromotionResponse{
		ID:                          promotion.ID,
		SellerID:                    promotion.SellerID,
		Name:                        promotion.Name,
		DisplayName:                 promotion.DisplayName,
		Slug:                        promotion.Slug,
		Description:                 promotion.Description,
		PromotionType:               promotion.PromotionType,
		DiscountConfig:              convertedConfig,
		AppliesTo:                   promotion.AppliesTo,
		EligibleFor:                 promotion.EligibleFor,
		CustomerSegmentID:           promotion.CustomerSegmentID,
		UsageLimitTotal:             promotion.UsageLimitTotal,
		UsageLimitPerCustomer:       promotion.UsageLimitPerCustomer,
		CurrentUsageCount:           promotion.CurrentUsageCount,
		AutoStart:                   promotion.AutoStart,
		AutoEnd:                     promotion.AutoEnd,
		Status:                      promotion.Status,
		CanStackWithOtherPromotions: promotion.CanStackWithOtherPromotions,
		CanStackWithCoupons:         promotion.CanStackWithCoupons,
		ShowOnStorefront:            promotion.ShowOnStorefront,
		BadgeText:                   promotion.BadgeText,
		BadgeColor:                  promotion.BadgeColor,
		Priority:                    promotion.Priority,
		SaleID:                      promotion.SaleID,
		CreatedAt:                   promotion.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                   promotion.UpdatedAt.Format(time.RFC3339),
	}

	// Expose thresholds at the top level from the converted config.
	if m, ok := convertedConfig["min_purchase_amount"].(commonModel.Money); ok {
		response.MinPurchaseAmount = &m
	}
	if m, ok := convertedConfig["max_discount_amount"].(commonModel.Money); ok {
		response.MaxDiscountAmount = &m
	}
	// Format StartsAt
	if promotion.StartsAt != nil {
		startsAt := promotion.StartsAt.Format(time.RFC3339)
		response.StartsAt = &startsAt
	}

	// Format EndsAt
	if promotion.EndsAt != nil {
		endsAt := promotion.EndsAt.Format(time.RFC3339)
		response.EndsAt = &endsAt
	}

	return response
}

// ApplyUpdatePromotionRequest applies non-nil fields from UpdatePromotionRequest to an existing Promotion entity
func ApplyUpdatePromotionRequest(
	existing *entity.Promotion,
	req model.UpdatePromotionRequest,
) *entity.Promotion {
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.DisplayName != nil {
		existing.DisplayName = req.DisplayName
	}
	if req.Slug != nil {
		existing.Slug = req.Slug
	}
	if req.Description != nil {
		existing.Description = req.Description
	}
	if req.PromotionType != nil {
		existing.PromotionType = *req.PromotionType
	}
	if req.DiscountConfig != nil {
		existing.DiscountConfig = db.JSONMap(*req.DiscountConfig)
	}
	if req.AppliesTo != nil {
		existing.AppliesTo = *req.AppliesTo
	}
	if req.EligibleFor != nil {
		existing.EligibleFor = *req.EligibleFor
	}
	if req.CustomerSegmentID != nil {
		existing.CustomerSegmentID = req.CustomerSegmentID
	}
	if req.UsageLimitTotal != nil {
		existing.UsageLimitTotal = req.UsageLimitTotal
	}
	if req.UsageLimitPerCustomer != nil {
		existing.UsageLimitPerCustomer = req.UsageLimitPerCustomer
	}
	if req.AutoStart != nil {
		existing.AutoStart = req.AutoStart
	}
	if req.AutoEnd != nil {
		existing.AutoEnd = req.AutoEnd
	}
	// NOTE: Status is intentionally NOT mapped here.
	// Status changes must go through the dedicated UpdateStatus API
	// to enforce the state machine transition rules.
	if req.CanStackWithOtherPromotions != nil {
		existing.CanStackWithOtherPromotions = req.CanStackWithOtherPromotions
	}
	if req.CanStackWithCoupons != nil {
		existing.CanStackWithCoupons = req.CanStackWithCoupons
	}
	if req.ShowOnStorefront != nil {
		existing.ShowOnStorefront = req.ShowOnStorefront
	}
	if req.BadgeText != nil {
		existing.BadgeText = req.BadgeText
	}
	if req.BadgeColor != nil {
		existing.BadgeColor = req.BadgeColor
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}
	if req.SaleID != nil {
		existing.SaleID = req.SaleID
	}

	if req.StartsAt != nil {
		if startsAt, err := time.Parse(time.RFC3339, *req.StartsAt); err == nil {
			existing.StartsAt = &startsAt
		}
	}
	if req.EndsAt != nil {
		if *req.EndsAt == "" {
			existing.EndsAt = nil
		} else if endsAt, err := time.Parse(time.RFC3339, *req.EndsAt); err == nil {
			existing.EndsAt = &endsAt
		}
	}

	return existing
}

// ConstructAppliedPromotionSummaryFromCartRequest creates an AppliedPromotionSummary from a CartValidationRequest
func ConstructAppliedPromotionSummaryFromCartRequest(
	cart *model.CartValidationRequest,
) *model.AppliedPromotionSummary {
	items := make([]model.CartItemSummary, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = model.CartItemSummary{
			ItemID:                 item.ItemID,
			ProductID:              item.ProductID,
			VariantID:              item.VariantID,
			Quantity:               item.Quantity,
			OriginalUnitPriceCents: item.PriceCents,
			FinalPriceCents:        item.TotalCents, // Initial final price is same as original; will be reduced as promotions are applied
			TotalDiscountCents:     0,
			AppliedPromotions:      []model.ItemPromotionDetail{}, // Initial total discount is 0; will be updated as promotions are applied
		}
	}

	return &model.AppliedPromotionSummary{
		Items:              items,
		AppliedPromotions:  []model.PromotionValidationResult{},
		SkippedPromotions:  []model.SkippedPromotionResult{},
		ShippingDiscount:   0,
		OriginalSubtotal:   cart.SubtotalCents,
		FinalSubtotal:      cart.SubtotalCents, // Initial final subtotal is same as original; will be reduced as promotions are applied
		TotalDiscountCents: 0,                  // Initial final total is subtotal + shipping; will be reduced as promotions are applied
	}
}
