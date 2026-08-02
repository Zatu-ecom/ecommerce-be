package model

import (
	"ecommerce-be/common"
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
	Value                        int64                  `json:"value"`
	MaxDiscountAmountCents       *int64                 `json:"maxDiscountAmountCents,omitempty"`
	AppliesTo                    entity.ScopeType       `json:"appliesTo"`
	MinPurchaseAmountCents       *int64                 `json:"minPurchaseAmountCents,omitempty"`
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
	DiscountCodes []DiscountCodeResponse    `json:"discountCodes"`
	Pagination    common.PaginationResponse `json:"pagination"`
}
