package entity

import (
	"time"

	"ecommerce-be/common/db"

	"gorm.io/gorm"
)

type DiscountType string

const (
	DiscountPercentage   DiscountType = "percentage"
	DiscountFixedAmount  DiscountType = "fixed_amount"
	DiscountFreeShipping DiscountType = "free_shipping"
	DiscountBuyXGetY     DiscountType = "buy_x_get_y"
)

// DiscountCode represents a coupon/discount code created by a seller
type DiscountCode struct {
	db.BaseEntity

	// Owner
	SellerID uint `json:"sellerId" gorm:"column:seller_id;not null;index"`

	// Code
	Code  string  `json:"code"  gorm:"column:code;size:50;not null"`
	Title *string `json:"title" gorm:"column:title;size:255"`

	// Discount Type
	DiscountType DiscountType `json:"discountType" gorm:"column:discount_type;size:50;not null"`

	// Discount Value
	Value int64 `json:"value" gorm:"column:value;not null"`

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
	CurrentUsageCount     int  `json:"currentUsageCount"     gorm:"column:current_usage_count;default:0"`

	// Combinations
	CanCombineWithOtherDiscounts *bool `json:"canCombineWithOtherDiscounts" gorm:"column:can_combine_with_other_discounts;default:false"`

	// Date Range
	StartsAt *time.Time `json:"startsAt" gorm:"column:starts_at;not null"`
	EndsAt   *time.Time `json:"endsAt"   gorm:"column:ends_at"`

	// Status
	IsActive *bool `json:"isActive" gorm:"column:is_active;default:true;index"`

	// Metadata
	Metadata JSONMap `json:"metadata" gorm:"column:metadata;type:jsonb;default:'{}'"`

	// Relationships
	Products    []DiscountCodeProduct    `json:"products,omitempty"    gorm:"foreignKey:DiscountCodeID"`
	Categories  []DiscountCodeCategory   `json:"categories,omitempty"  gorm:"foreignKey:DiscountCodeID"`
	Collections []DiscountCodeCollection `json:"collections,omitempty" gorm:"foreignKey:DiscountCodeID"`
	Usages      []DiscountCodeUsage      `json:"usages,omitempty"      gorm:"foreignKey:DiscountCodeID"`
}

// TableName specifies the table name
func (DiscountCode) TableName() string {
	return "discount_code"
}

// BeforeCreate hook
func (dc *DiscountCode) BeforeCreate(tx *gorm.DB) error {
	// Code validation can be added here
	// For example, ensure code is uppercase, alphanumeric, etc.
	return nil
}
