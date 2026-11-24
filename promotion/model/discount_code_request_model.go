package model

import (
	commonModel "ecommerce-be/common/model"
	"ecommerce-be/promotion/entity"
)

// CreateDiscountCodeRequest represents the request body for creating a discount code
type CreateDiscountCodeRequest struct {
	Code                         string                 `json:"code" binding:"required,min=1,max=50"`
	Title                        *string                `json:"title" binding:"omitempty,max=255"`
	Description                  *string                `json:"description" binding:"omitempty"`
	DiscountType                 entity.DiscountType    `json:"discountType" binding:"required,oneof=percentage fixed_amount free_shipping buy_x_get_y"`
	Value                        float64                `json:"value"` // major units for fixed_amount; plain percentage for percentage
	MaxDiscountAmount            *float64               `json:"maxDiscountAmount" binding:"omitempty,min=0"`
	AppliesTo                    entity.ScopeType       `json:"appliesTo" binding:"required,oneof=all_products specific_products specific_categories specific_collections specific_variant"`
	MinPurchaseAmount            *float64               `json:"minPurchaseAmount" binding:"omitempty,min=0"`
	MinQuantity                  *int                   `json:"minQuantity" binding:"omitempty,min=1"`
	CustomerEligibility          entity.EligibilityType `json:"customerEligibility" binding:"omitempty,oneof=everyone new_customers specific_segment"`
	CustomerSegmentID            *uint                  `json:"customerSegmentId" binding:"omitempty"`
	UsageLimitTotal              *int                   `json:"usageLimitTotal" binding:"omitempty,min=1"`
	UsageLimitPerCustomer        *int                   `json:"usageLimitPerCustomer" binding:"omitempty,min=1"`
	UsageResetTimeType           entity.ResetTimeType   `json:"usageResetTimeType" binding:"omitempty,oneof=none day week month year"`
	UsageResetAmount             *int                   `json:"usageResetAmount" binding:"omitempty,min=1"`
	CanCombineWithOtherDiscounts *bool                  `json:"canCombineWithOtherDiscounts"`
	StartsAt                     string                 `json:"startsAt" binding:"required"`
	EndsAt                       *string                `json:"endsAt" binding:"omitempty"`
	IsActive                     *bool                  `json:"isActive"`
	AutoStart                    *bool                  `json:"autoStart" binding:"omitempty"`
	AutoEnd                      *bool                  `json:"autoEnd" binding:"omitempty"`
	Metadata                     map[string]any         `json:"metadata"`
}

// UpdateDiscountCodeRequest represents a partial update (code is immutable after create)
type UpdateDiscountCodeRequest struct {
	Title                        *string                 `json:"title" binding:"omitempty,max=255"`
	Description                  *string                 `json:"description" binding:"omitempty"`
	DiscountType                 *entity.DiscountType    `json:"discountType" binding:"omitempty,oneof=percentage fixed_amount free_shipping buy_x_get_y"`
	Value                        *float64                `json:"value" binding:"omitempty"` // major units for fixed_amount; plain percentage for percentage
	MaxDiscountAmount            *float64                `json:"maxDiscountAmount" binding:"omitempty,min=0"`
	AppliesTo                    *entity.ScopeType       `json:"appliesTo" binding:"omitempty,oneof=all_products specific_products specific_categories specific_collections specific_variant"`
	MinPurchaseAmount            *float64                `json:"minPurchaseAmount" binding:"omitempty,min=0"`
	MinQuantity                  *int                    `json:"minQuantity" binding:"omitempty,min=1"`
	CustomerEligibility          *entity.EligibilityType `json:"customerEligibility" binding:"omitempty,oneof=everyone new_customers specific_segment"`
	CustomerSegmentID            *uint                   `json:"customerSegmentId" binding:"omitempty"`
	UsageLimitTotal              *int                    `json:"usageLimitTotal" binding:"omitempty,min=1"`
	UsageLimitPerCustomer        *int                    `json:"usageLimitPerCustomer" binding:"omitempty,min=1"`
	UsageResetTimeType           *entity.ResetTimeType   `json:"usageResetTimeType" binding:"omitempty,oneof=none day week month year"`
	UsageResetAmount             *int                    `json:"usageResetAmount" binding:"omitempty,min=1"`
	CanCombineWithOtherDiscounts *bool                   `json:"canCombineWithOtherDiscounts"`
	StartsAt                     *string                 `json:"startsAt" binding:"omitempty"`
	EndsAt                       *string                 `json:"endsAt" binding:"omitempty"`
	IsActive                     *bool                   `json:"isActive"`
	AutoStart                    *bool                   `json:"autoStart" binding:"omitempty"`
	AutoEnd                      *bool                   `json:"autoEnd" binding:"omitempty"`
	Metadata                     *map[string]any         `json:"metadata"`
}

// UpdateDiscountCodeStatusRequest toggles isActive only
type UpdateDiscountCodeStatusRequest struct {
	IsActive bool `json:"isActive"`
}

// ListDiscountCodesRequest represents query parameters for listing discount codes
type ListDiscountCodesRequest struct {
	commonModel.BaseListParams
	SellerID     uint
	IsActive     *bool                `form:"isActive"`
	DiscountType *entity.DiscountType `form:"discountType"`
	AppliesTo    *entity.ScopeType    `form:"appliesTo"`
}
