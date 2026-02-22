package factory

import (
	"time"

	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
)

// PromotionRequestToEntity converts CreatePromotionRequest to Promotion entity
func PromotionRequestToEntity(req model.CreatePromotionRequest, sellerID uint) *entity.Promotion {
	promotion := &entity.Promotion{
		SellerID:       sellerID,
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		Slug:           req.Slug,
		Description:    req.Description,
		PromotionType:  req.PromotionType,
		DiscountConfig: entity.DiscountConfig(req.DiscountConfig),
		AppliesTo:      req.AppliesTo,
		MinPurchaseAmountCents: req.MinPurchaseAmountCents,
		MinQuantity:            req.MinQuantity,
		MaxDiscountAmountCents: req.MaxDiscountAmountCents,
		EligibleFor:       req.EligibleFor,
		CustomerSegmentID: req.CustomerSegmentID,
		UsageLimitTotal:       req.UsageLimitTotal,
		UsageLimitPerCustomer: req.UsageLimitPerCustomer,
		AutoStart: req.AutoStart,
		AutoEnd:   req.AutoEnd,
		CanStackWithOtherPromotions: req.CanStackWithOtherPromotions,
		CanStackWithCoupons:         req.CanStackWithCoupons,
		ShowOnStorefront: req.ShowOnStorefront,
		BadgeText:        req.BadgeText,
		BadgeColor:       req.BadgeColor,
		SaleID:           req.SaleID,
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

	// Set Metadata
	if req.Metadata != nil {
		promotion.Metadata = entity.JSONMap(req.Metadata)
	}

	return promotion
}

// PromotionEntityToResponse converts Promotion entity to PromotionResponse
func PromotionEntityToResponse(promotion *entity.Promotion) *model.PromotionResponse {
	response := &model.PromotionResponse{
		ID:                 promotion.ID,
		SellerID:           promotion.SellerID,
		Name:               promotion.Name,
		DisplayName:        promotion.DisplayName,
		Slug:               promotion.Slug,
		Description:        promotion.Description,
		PromotionType:      promotion.PromotionType,
		DiscountConfig:     map[string]interface{}(promotion.DiscountConfig),
		AppliesTo:          promotion.AppliesTo,
		MinPurchaseAmountCents: promotion.MinPurchaseAmountCents,
		MinQuantity:            promotion.MinQuantity,
		MaxDiscountAmountCents: promotion.MaxDiscountAmountCents,
		EligibleFor:       promotion.EligibleFor,
		CustomerSegmentID: promotion.CustomerSegmentID,
		UsageLimitTotal:       promotion.UsageLimitTotal,
		UsageLimitPerCustomer: promotion.UsageLimitPerCustomer,
		CurrentUsageCount:     promotion.CurrentUsageCount,
		AutoStart: promotion.AutoStart,
		AutoEnd:   promotion.AutoEnd,
		Status:    promotion.Status,
		CanStackWithOtherPromotions: promotion.CanStackWithOtherPromotions,
		CanStackWithCoupons:         promotion.CanStackWithCoupons,
		ShowOnStorefront: promotion.ShowOnStorefront,
		BadgeText:        promotion.BadgeText,
		BadgeColor:       promotion.BadgeColor,
		Priority:         promotion.Priority,
		Metadata:         map[string]interface{}(promotion.Metadata),
		SaleID:           promotion.SaleID,
		CreatedAt:        promotion.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        promotion.UpdatedAt.Format(time.RFC3339),
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
