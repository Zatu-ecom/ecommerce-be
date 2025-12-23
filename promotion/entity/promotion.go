package entity

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"ecommerce-be/common/db"

	"gorm.io/gorm"
)

type PromotionType string

const (
	PromoTypePercentage   PromotionType = "percentage_discount"
	PromoTypeFixedAmount  PromotionType = "fixed_amount"
	PromoTypeBuyXGetY     PromotionType = "buy_x_get_y"
	PromoTypeFreeShipping PromotionType = "free_shipping"
	PromoTypeBundle       PromotionType = "bundle"
	PromoTypeTiered       PromotionType = "tiered"
	PromoTypeFlashSale    PromotionType = "flash_sale"
)

type ScopeType string

const (
	ScopeAllProducts        ScopeType = "all_products"
	ScopeSpecificProducts   ScopeType = "specific_products"
	ScopeSpecificCategories ScopeType = "specific_categories"
	ScopeSpecificCollections ScopeType = "specific_collections"
)

type EligibilityType string

const (
	EligibleEveryone           EligibilityType = "everyone"
	EligibleNewCustomers       EligibilityType = "new_customers"
	EligibleReturningCustomers EligibilityType = "returning_customers"
	EligibleSpecificSegment    EligibilityType = "specific_segment"
)

type PromotionStatus string

const (
	StatusDraft     PromotionStatus = "draft"
	StatusScheduled PromotionStatus = "scheduled"
	StatusActive    PromotionStatus = "active"
	StatusPaused    PromotionStatus = "paused"
	StatusEnded     PromotionStatus = "ended"
	StatusCancelled PromotionStatus = "cancelled"
)

// Promotion represents a seller-created sale or promotional offer
type Promotion struct {
	db.BaseEntity

	// Owner
	SellerID uint `json:"sellerId" gorm:"column:seller_id;not null;index"`

	// Promotion Info
	Name        string  `json:"name"        gorm:"column:name;size:255;not null"`
	DisplayName *string `json:"displayName" gorm:"column:display_name;size:255"`
	Slug        *string `json:"slug"        gorm:"column:slug;size:255"`
	Description *string `json:"description" gorm:"column:description;type:text"`

	// Promotion Mechanics
	PromotionType  PromotionType  `json:"promotionType"  gorm:"column:promotion_type;size:50;not null"`
	DiscountConfig DiscountConfig `json:"discountConfig" gorm:"column:discount_config;type:jsonb;not null"`

	// Scope
	AppliesTo ScopeType `json:"appliesTo" gorm:"column:applies_to;size:50;not null;default:specific_products"`

	// Conditions
	MinPurchaseAmountCents *int64 `json:"minPurchaseAmountCents" gorm:"column:min_purchase_amount_cents;default:0"`
	MinQuantity            *int   `json:"minQuantity"            gorm:"column:min_quantity;default:1"`
	MaxDiscountAmountCents *int64 `json:"maxDiscountAmountCents" gorm:"column:max_discount_amount_cents"`

	// Customer Eligibility
	EligibleFor       EligibilityType `json:"eligibleFor"       gorm:"column:eligible_for;size:50;default:everyone"`
	CustomerSegmentID *uint           `json:"customerSegmentId" gorm:"column:customer_segment_id;index"`

	// Usage Limits
	UsageLimitTotal       *int `json:"usageLimitTotal"       gorm:"column:usage_limit_total"`
	UsageLimitPerCustomer *int `json:"usageLimitPerCustomer" gorm:"column:usage_limit_per_customer;default:1"`
	CurrentUsageCount     int  `json:"currentUsageCount"     gorm:"column:current_usage_count;default:0"`

	// Date Range
	StartsAt *time.Time `json:"startsAt" gorm:"column:starts_at;not null"`
	EndsAt   *time.Time `json:"endsAt"   gorm:"column:ends_at"`

	// Automatic Start/Stop
	AutoStart *bool `json:"autoStart" gorm:"column:auto_start;default:true"`
	AutoEnd   *bool `json:"autoEnd"   gorm:"column:auto_end;default:true"`

	// Status
	Status PromotionStatus `json:"status" gorm:"column:status;size:50;default:draft;index"`

	// Stacking Rules
	CanStackWithOtherPromotions *bool `json:"canStackWithOtherPromotions" gorm:"column:can_stack_with_other_promotions;default:false"`
	CanStackWithCoupons         *bool `json:"canStackWithCoupons"         gorm:"column:can_stack_with_coupons;default:true"`

	// Display Settings
	ShowOnStorefront *bool   `json:"showOnStorefront" gorm:"column:show_on_storefront;default:true"`
	BadgeText        *string `json:"badgeText"        gorm:"column:badge_text;size:50"`
	BadgeColor       *string `json:"badgeColor"       gorm:"column:badge_color;size:20"`

	// Priority
	Priority int `json:"priority" gorm:"column:priority;default:0"`

	// Metadata
	Metadata JSONMap `json:"metadata" gorm:"column:metadata;type:jsonb;default:'{}'"`

	// Relationships
	Products    []PromotionProduct    `json:"products,omitempty"    gorm:"foreignKey:PromotionID"`
	Categories  []PromotionCategory   `json:"categories,omitempty"  gorm:"foreignKey:PromotionID"`
	Collections []PromotionCollection `json:"collections,omitempty" gorm:"foreignKey:PromotionID"`
	Usages      []PromotionUsage      `json:"usages,omitempty"      gorm:"foreignKey:PromotionID"`
}

// DiscountConfig holds the discount configuration in JSONB format
type DiscountConfig map[string]interface{}

// Scan implements sql.Scanner interface for reading from database
func (dc *DiscountConfig) Scan(value interface{}) error {
	if value == nil {
		*dc = make(DiscountConfig)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, dc)
}

// Value implements driver.Valuer interface for writing to database
func (dc DiscountConfig) Value() (driver.Value, error) {
	if dc == nil {
		return json.Marshal(make(map[string]interface{}))
	}
	return json.Marshal(dc)
}

// JSONMap represents a JSONB field
type JSONMap map[string]interface{}

// Scan implements sql.Scanner interface
func (jm *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*jm = make(JSONMap)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, jm)
}

// Value implements driver.Valuer interface
func (jm JSONMap) Value() (driver.Value, error) {
	if jm == nil {
		return json.Marshal(make(map[string]interface{}))
	}
	return json.Marshal(jm)
}

// TableName specifies the table name
func (Promotion) TableName() string {
	return "promotion"
}

// BeforeCreate hook
func (p *Promotion) BeforeCreate(tx *gorm.DB) error {
	// Generate slug if not provided
	if p.Slug == nil || *p.Slug == "" {
		// Simple slug generation (can be improved)
		slug := generateSlug(p.Name)
		p.Slug = &slug
	}
	return nil
}

// Helper function to generate slug (basic implementation)
func generateSlug(name string) string {
	// This is a simple implementation
	// In production, use a proper slug generation library
	return name
}
