package model

// PercentageDiscountConfig represents the configuration for percentage_discount promotion type
type PercentageDiscountConfig struct {
	Percentage       float64 `json:"percentage"                   binding:"required,min=0.01,max=100"`
	MaxDiscountCents *int64  `json:"max_discount_cents,omitempty" binding:"omitempty,min=0"`
	MinOrderCents    *int64  `json:"min_order_cents,omitempty"    binding:"omitempty,min=0"`
}

// FixedAmountConfig represents the configuration for fixed_amount promotion type
type FixedAmountConfig struct {
	AmountCents   int64  `json:"amount_cents"              binding:"required,min=1"`
	MinOrderCents *int64 `json:"min_order_cents,omitempty" binding:"omitempty,min=0"`
}

// BuyXGetYScopeType defines how same-reward Buy X Get Y should match items.
type BuyXGetYScopeType string

const (
	BuyXGetYScopeSameVariant  BuyXGetYScopeType = "same_variant"
	BuyXGetYScopeSameProduct  BuyXGetYScopeType = "same_product"
	BuyXGetYScopeSameCategory BuyXGetYScopeType = "same_category"
)

// BuyXGetYConfig represents the configuration for buy_x_get_y promotion type.
// Defaults:
//   - is_same_reward: true
//   - scope_type: same_product (when is_same_reward = true)
type BuyXGetYConfig struct {
	BuyQuantity  int               `json:"buy_quantity"             binding:"required,min=1"`  // Number of items to buy
	GetQuantity  int               `json:"get_quantity"             binding:"required,min=1"`  // Number of items to get free
	MaxSets      *int              `json:"max_sets,omitempty"       binding:"omitempty,min=1"` // Optional: limit number of sets per customer
	IsSameReward *bool             `json:"is_same_reward,omitempty"`                           // True => reward comes from the same target pool (defaults to true)
	ScopeType    BuyXGetYScopeType `json:"scope_type,omitempty"     binding:"omitempty,oneof=same_variant same_product same_category"`
	GetProductID *uint             `json:"get_product_id,omitempty" binding:"omitempty"`
}

// FreeShippingConfig represents the configuration for free_shipping promotion type
type FreeShippingConfig struct {
	MinOrderCents            *int64 `json:"min_order_cents,omitempty"             binding:"omitempty,min=0"`
	MaxShippingDiscountCents *int64 `json:"max_shipping_discount_cents,omitempty" binding:"omitempty,min=0"`
}

// BundleItemConfig represents a single item in a bundle
type BundleItemConfig struct {
	ProductID uint  `json:"product_id"           binding:"required"`
	VariantID *uint `json:"variant_id,omitempty"`
	Quantity  int   `json:"quantity"             binding:"required,min=1"`
}

// BundleConfig represents the configuration for bundle promotion type
type BundleConfig struct {
	BundleItems         []BundleItemConfig `json:"bundle_items"                    binding:"required,min=1,dive"`
	BundleDiscountType  string             `json:"bundle_discount_type"            binding:"required,oneof=percentage fixed_amount fixed_price"`
	BundleDiscountValue *float64           `json:"bundle_discount_value,omitempty" binding:"omitempty,min=0"`
	BundlePriceCents    *int64             `json:"bundle_price_cents,omitempty"    binding:"omitempty,min=0"`
}

// TierConfig represents a single tier in tiered pricing
type TierConfig struct {
	Min           int     `json:"min"            binding:"required,min=0"`
	Max           *int    `json:"max,omitempty"  binding:"omitempty,min=1"`
	DiscountType  string  `json:"discount_type"  binding:"required,oneof=percentage fixed_amount"`
	DiscountValue float64 `json:"discount_value" binding:"required,min=0"`
}

// TieredConfig represents the configuration for tiered promotion type
type TieredConfig struct {
	TierType string       `json:"tier_type" binding:"required,oneof=quantity spend"`
	Tiers    []TierConfig `json:"tiers"     binding:"required,min=1,dive"`
}

// FlashSaleConfig represents the configuration for flash_sale promotion type
type FlashSaleConfig struct {
	DiscountType     string  `json:"discount_type"                binding:"required,oneof=percentage fixed_amount"`
	DiscountValue    float64 `json:"discount_value"               binding:"required,min=0"`
	MaxDiscountCents *int64  `json:"max_discount_cents,omitempty" binding:"omitempty,min=0"`
	StockLimit       *int    `json:"stock_limit,omitempty"        binding:"omitempty,min=1"`
	SoldCount        *int    `json:"sold_count,omitempty"         binding:"omitempty,min=0"`
}
