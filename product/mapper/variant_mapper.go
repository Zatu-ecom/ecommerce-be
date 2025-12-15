package mapper

import "ecommerce-be/product/entity"

// VariantAggregation represents aggregated variant data for a product
type VariantAggregation struct {
	HasVariants   bool
	TotalVariants int
	MinPrice      float64
	MaxPrice      float64
	AllowPurchase bool // At least one variant allows purchase
	MainImage     string
	OptionNames   []string
	OptionValues  map[string][]string // optionName -> []values
}

// VariantWithOptions represents a variant with its selected option values
type VariantWithOptions struct {
	Variant         entity.ProductVariant
	SelectedOptions []SelectedOptionValue
}

// SelectedOptionValue represents a selected option value for a variant
type SelectedOptionValue struct {
	OptionID          uint
	OptionName        string
	OptionDisplayName string
	ValueID           uint
	Value             string
	ValueDisplayName  string
	ColorCode         string
}

// Batch query to get all variant option values with option and option value details
// This is a single JOIN query for performance
type OptionValueData struct {
	VariantID         uint
	OptionID          uint
	OptionName        string
	OptionDisplayName string
	ValueID           uint
	Value             string
	ValueDisplayName  string
	ColorCode         string
}

// VariantBasicInfoRow represents basic product info with variant ID for cross-service queries
// Used by inventory service to map variant IDs to product details
// Returns flat rows for efficient in-memory grouping by product
type VariantBasicInfoRow struct {
	VariantID   uint
	ProductID   uint
	ProductName string
	CategoryID  uint
	BaseSKU     string
	SellerID    uint
}
