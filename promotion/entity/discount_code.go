package entity

import (
	"time"

	"ecommerce-be/common/db"
)

type DiscountType string

const (
	DiscountPercentage   DiscountType = "percentage"
	DiscountFixedAmount  DiscountType = "fixed_amount"
	DiscountFreeShipping DiscountType = "free_shipping"
	DiscountBuyXGetY     DiscountType = "buy_x_get_y"
)

// ResetTimeType defines the time unit for usage reset period
type ResetTimeType string

const (
	ResetTimeTypeNone  ResetTimeType = "none"  // No reset (lifetime limit)
	ResetTimeTypeDay   ResetTimeType = "day"   // Reset every X days
	ResetTimeTypeWeek  ResetTimeType = "week"  // Reset every X weeks
	ResetTimeTypeMonth ResetTimeType = "month" // Reset every X months
	ResetTimeTypeYear  ResetTimeType = "year"  // Reset every X years
)

// DiscountCode represents a coupon/discount code created by a seller
type DiscountCode struct {
	db.BaseEntity

	// Owner
	SellerID uint `json:"sellerId" gorm:"column:seller_id;not null;index"`

	// Code
	Code        string  `json:"code"        gorm:"column:code;size:50;not null"`
	Title       *string `json:"title"       gorm:"column:title;size:255"`
	Description *string `json:"description" gorm:"column:description;type:text"`

	// Discount Type
	DiscountType DiscountType `json:"discountType" gorm:"column:discount_type;size:50;not null"`

	// Discount Value
	// For percentage: value = 15 means 15%
	// For fixed_amount: value = 10000 means ₹100 (in cents/paise)
	Value int64 `json:"value" gorm:"column:value;not null"`

	// Maximum Discount Cap (for percentage discounts)
	// If set, caps the discount amount even if percentage would give more
	// Example: 15% discount with MaxDiscountAmountCents=10000 means max ₹100 discount
	MaxDiscountAmountCents *int64 `json:"maxDiscountAmountCents" gorm:"column:max_discount_amount_cents"`

	// Applies To
	AppliesTo ScopeType `json:"appliesTo" gorm:"column:applies_to;size:50;not null;default:all_products"`

	// Requirements
	MinPurchaseAmountCents *int64 `json:"minPurchaseAmountCents" gorm:"column:min_purchase_amount_cents"`
	MinQuantity            *int   `json:"minQuantity"            gorm:"column:min_quantity"`

	// Customer Eligibility
	CustomerEligibility EligibilityType `json:"customerEligibility" gorm:"column:customer_eligibility;size:50;default:everyone"`
	CustomerSegmentID   *uint           `json:"customerSegmentId"   gorm:"column:customer_segment_id;index"`

	// Usage Limits
	UsageLimitTotal       *int `json:"usageLimitTotal"       gorm:"column:usage_limit_total"`
	UsageLimitPerCustomer *int `json:"usageLimitPerCustomer" gorm:"column:usage_limit_per_customer;default:1"`

	// Usage Reset Configuration (e.g., reset every 1 month, 2 weeks, 30 days)
	// If ResetTimeType is "none", UsageLimitPerCustomer is lifetime limit
	UsageResetTimeType ResetTimeType `json:"usageResetTimeType" gorm:"column:usage_reset_time_type;size:20;default:'none'"`
	UsageResetAmount   *int          `json:"usageResetAmount"   gorm:"column:usage_reset_amount"`

	// Combinations
	CanCombineWithOtherDiscounts *bool `json:"canCombineWithOtherDiscounts" gorm:"column:can_combine_with_other_discounts;default:false"`

	// Date Range
	StartsAt *time.Time `json:"startsAt" gorm:"column:starts_at;not null"`
	EndsAt   *time.Time `json:"endsAt"   gorm:"column:ends_at"`

	// Status
	IsActive *bool `json:"isActive" gorm:"column:is_active;default:true;index"`

	// Metadata
	Metadata JSONMap `json:"metadata" gorm:"column:metadata;type:jsonb;default:'{}'"`
}
