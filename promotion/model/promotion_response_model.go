package model

import (
	"ecommerce-be/common"
	"ecommerce-be/promotion/entity"
)

// PromotionResponse represents the promotion data returned in API responses
type PromotionResponse struct {
	ID uint `json:"id"`

	// Owner
	SellerID uint `json:"sellerId"`

	// Promotion Info
	Name        string  `json:"name"`
	DisplayName *string `json:"displayName,omitempty"`
	Slug        *string `json:"slug,omitempty"`
	Description *string `json:"description,omitempty"`

	// Promotion Mechanics
	PromotionType  entity.PromotionType   `json:"promotionType"`
	DiscountConfig map[string]interface{} `json:"discountConfig"`

	// Scope
	AppliesTo entity.ScopeType `json:"appliesTo"`

	// Conditions
	MinPurchaseAmountCents *int64 `json:"minPurchaseAmountCents,omitempty"`
	MinQuantity            *int   `json:"minQuantity,omitempty"`
	MaxDiscountAmountCents *int64 `json:"maxDiscountAmountCents,omitempty"`

	// Customer Eligibility
	EligibleFor       entity.EligibilityType `json:"eligibleFor"`
	CustomerSegmentID *uint                  `json:"customerSegmentId,omitempty"`

	// Usage Limits
	UsageLimitTotal       *int `json:"usageLimitTotal,omitempty"`
	UsageLimitPerCustomer *int `json:"usageLimitPerCustomer,omitempty"`
	CurrentUsageCount     int  `json:"currentUsageCount"`

	// Date Range
	StartsAt *string `json:"startsAt,omitempty"`
	EndsAt   *string `json:"endsAt,omitempty"`

	// Automatic Start/Stop
	AutoStart *bool `json:"autoStart,omitempty"`
	AutoEnd   *bool `json:"autoEnd,omitempty"`

	// Status
	Status entity.CampaignStatus `json:"status"`

	// Stacking Rules
	CanStackWithOtherPromotions *bool `json:"canStackWithOtherPromotions,omitempty"`
	CanStackWithCoupons         *bool `json:"canStackWithCoupons,omitempty"`

	// Display Settings
	ShowOnStorefront *bool   `json:"showOnStorefront,omitempty"`
	BadgeText        *string `json:"badgeText,omitempty"`
	BadgeColor       *string `json:"badgeColor,omitempty"`

	// Priority
	Priority int `json:"priority"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Sale
	SaleID *uint `json:"saleId,omitempty"`

	// Timestamps
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`

	// Relationships (optional)
	Products    []PromotionProductResponse    `json:"products,omitempty"`
	Categories  []PromotionCategoryResponse   `json:"categories,omitempty"`
	Collections []PromotionCollectionResponse `json:"collections,omitempty"`
}

// ListPromotionsResponse represents the paginated response for listing promotions
type ListPromotionsResponse struct {
	Promotions []*PromotionResponse   `json:"promotions"`
	Pagination common.PaginationResponse `json:"pagination"`
}
