package factory

import (
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
)

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
func PromotionEntityToResponse(promotion *entity.Promotion) *model.PromotionResponse {
	response := &model.PromotionResponse{
		ID:                          promotion.ID,
		SellerID:                    promotion.SellerID,
		Name:                        promotion.Name,
		DisplayName:                 promotion.DisplayName,
		Slug:                        promotion.Slug,
		Description:                 promotion.Description,
		PromotionType:               promotion.PromotionType,
		DiscountConfig:              map[string]interface{}(promotion.DiscountConfig),
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
