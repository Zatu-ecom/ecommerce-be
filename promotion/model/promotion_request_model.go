package model

import (
	"ecommerce-be/common"
	"ecommerce-be/promotion/entity"
)

// CreatePromotionRequest represents the request body for creating a promotion
type CreatePromotionRequest struct {
	// Basic Info
	Name        string  `json:"name"        binding:"required,min=3,max=255"`
	DisplayName *string `json:"displayName" binding:"omitempty,max=255"`
	Slug        *string `json:"slug"        binding:"omitempty,max=255"`
	Description *string `json:"description" binding:"omitempty"`

	// Promotion Mechanics
	PromotionType  entity.PromotionType   `json:"promotionType"  binding:"required,oneof=percentage_discount fixed_amount buy_x_get_y free_shipping bundle tiered flash_sale"`
	DiscountConfig map[string]interface{} `json:"discountConfig" binding:"required"`

	// Scope
	AppliesTo entity.ScopeType `json:"appliesTo" binding:"required,oneof=all_products specific_products specific_categories specific_collections"`

	// Conditions
	MinPurchaseAmountCents *int64 `json:"minPurchaseAmountCents" binding:"omitempty,min=0"`
	MinQuantity            *int   `json:"minQuantity"            binding:"omitempty,min=1"`
	MaxDiscountAmountCents *int64 `json:"maxDiscountAmountCents" binding:"omitempty,min=0"`

	// Customer Eligibility
	EligibleFor       entity.EligibilityType `json:"eligibleFor"       binding:"omitempty,oneof=everyone new_customers specific_segment"`
	CustomerSegmentID *uint                  `json:"customerSegmentId" binding:"omitempty"`

	// Usage Limits
	UsageLimitTotal       *int `json:"usageLimitTotal"       binding:"omitempty,min=1"`
	UsageLimitPerCustomer *int `json:"usageLimitPerCustomer" binding:"omitempty,min=1"`

	// Date Range
	StartsAt *string `json:"startsAt" binding:"required"`
	EndsAt   *string `json:"endsAt"   binding:"omitempty"`

	// Automatic Start/Stop
	AutoStart *bool `json:"autoStart" binding:"omitempty"`
	AutoEnd   *bool `json:"autoEnd"   binding:"omitempty"`

	// Status
	Status entity.CampaignStatus `json:"status" binding:"omitempty,oneof=draft scheduled active paused ended cancelled"`

	// Stacking Rules
	CanStackWithOtherPromotions *bool `json:"canStackWithOtherPromotions" binding:"omitempty"`
	CanStackWithCoupons         *bool `json:"canStackWithCoupons"         binding:"omitempty"`

	// Display Settings
	ShowOnStorefront *bool   `json:"showOnStorefront" binding:"omitempty"`
	BadgeText        *string `json:"badgeText"        binding:"omitempty,max=50"`
	BadgeColor       *string `json:"badgeColor"       binding:"omitempty,max=20"`

	// Priority
	Priority *int `json:"priority" binding:"omitempty"`

	// Sale
	SaleID *uint `json:"saleId" binding:"omitempty"`
}

// UpdatePromotionRequest represents the request body for updating a promotion
type UpdatePromotionRequest struct {
	Name        *string `json:"name"        binding:"omitempty,min=3,max=255"`
	DisplayName *string `json:"displayName" binding:"omitempty,max=255"`
	Slug        *string `json:"slug"        binding:"omitempty,max=255"`
	Description *string `json:"description" binding:"omitempty"`

	PromotionType  *entity.PromotionType   `json:"promotionType"  binding:"omitempty,oneof=percentage_discount fixed_amount buy_x_get_y free_shipping bundle tiered flash_sale"`
	DiscountConfig *map[string]interface{} `json:"discountConfig" binding:"omitempty"`

	AppliesTo *entity.ScopeType `json:"appliesTo" binding:"omitempty,oneof=all_products specific_products specific_categories specific_collections"`

	MinPurchaseAmountCents *int64 `json:"minPurchaseAmountCents" binding:"omitempty,min=0"`
	MinQuantity            *int   `json:"minQuantity"            binding:"omitempty,min=1"`
	MaxDiscountAmountCents *int64 `json:"maxDiscountAmountCents" binding:"omitempty,min=0"`

	EligibleFor       *entity.EligibilityType `json:"eligibleFor"       binding:"omitempty,oneof=everyone new_customers specific_segment"`
	CustomerSegmentID *uint                   `json:"customerSegmentId" binding:"omitempty"`

	UsageLimitTotal       *int `json:"usageLimitTotal"       binding:"omitempty,min=1"`
	UsageLimitPerCustomer *int `json:"usageLimitPerCustomer" binding:"omitempty,min=1"`

	StartsAt *string `json:"startsAt" binding:"omitempty"`
	EndsAt   *string `json:"endsAt"   binding:"omitempty"`

	AutoStart *bool `json:"autoStart" binding:"omitempty"`
	AutoEnd   *bool `json:"autoEnd"   binding:"omitempty"`

	CanStackWithOtherPromotions *bool `json:"canStackWithOtherPromotions" binding:"omitempty"`
	CanStackWithCoupons         *bool `json:"canStackWithCoupons"         binding:"omitempty"`

	ShowOnStorefront *bool   `json:"showOnStorefront" binding:"omitempty"`
	BadgeText        *string `json:"badgeText"        binding:"omitempty,max=50"`
	BadgeColor       *string `json:"badgeColor"       binding:"omitempty,max=20"`

	Priority *int `json:"priority" binding:"omitempty"`

	SaleID *uint `json:"saleId" binding:"omitempty"`
}

// UpdateStatusRequest represents the request body for status updates
type UpdateStatusRequest struct {
	Status entity.CampaignStatus `json:"status" binding:"required,oneof=draft scheduled active paused ended cancelled"`
}

// ListPromotionsRequest represents query parameters for listing promotions
type ListPromotionsRequest struct {
	common.BaseListParams
	SellerID      uint
	Status        *entity.CampaignStatus `form:"status"`
	PromotionType *entity.PromotionType  `form:"promotionType"`
	AppliesTo     *entity.ScopeType      `form:"appliesTo"`
}
