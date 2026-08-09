package model

import (
	commonModel "ecommerce-be/common/model"
	"ecommerce-be/promotion/entity"
)

// DiscountCodeResponse represents discount code data returned in API responses
type DiscountCodeResponse struct {
	ID                           uint                   `json:"id"`
	SellerID                     uint                   `json:"sellerId"`
	Code                         string                 `json:"code"`
	Title                        *string                `json:"title,omitempty"`
	Description                  *string                `json:"description,omitempty"`
	DiscountType                 entity.DiscountType    `json:"discountType"`
	Value                        any                    `json:"value"` // Money for fixed_amount; plain number for percentage
	MaxDiscountAmount            *commonModel.Money     `json:"maxDiscountAmount,omitempty"`
	AppliesTo                    entity.ScopeType       `json:"appliesTo"`
	MinPurchaseAmount            *commonModel.Money     `json:"minPurchaseAmount,omitempty"`
	MinQuantity                  *int                   `json:"minQuantity,omitempty"`
	CustomerEligibility          entity.EligibilityType `json:"customerEligibility"`
	CustomerSegmentID            *uint                  `json:"customerSegmentId,omitempty"`
	UsageLimitTotal              *int                   `json:"usageLimitTotal,omitempty"`
	UsageLimitPerCustomer        *int                   `json:"usageLimitPerCustomer,omitempty"`
	CurrentUsageCount            int                    `json:"currentUsageCount"`
	UsageResetTimeType           entity.ResetTimeType   `json:"usageResetTimeType"`
	UsageResetAmount             *int                   `json:"usageResetAmount,omitempty"`
	CanCombineWithOtherDiscounts *bool                  `json:"canCombineWithOtherDiscounts"`
	StartsAt                     string                 `json:"startsAt"`
	EndsAt                       *string                `json:"endsAt,omitempty"`
	IsActive                     bool                   `json:"isActive"`
	AutoStart                    *bool                  `json:"autoStart,omitempty"`
	AutoEnd                      *bool                  `json:"autoEnd,omitempty"`
	Metadata                     map[string]any         `json:"metadata,omitempty"`
	CreatedAt                    string                 `json:"createdAt"`
	UpdatedAt                    string                 `json:"updatedAt"`
}

// ListDiscountCodesResponse is the paginated list response for discount codes
type ListDiscountCodesResponse struct {
	DiscountCodes []DiscountCodeResponse         `json:"discountCodes"`
	Pagination    commonModel.PaginationResponse `json:"pagination"`
}
