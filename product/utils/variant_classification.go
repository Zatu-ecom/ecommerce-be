package utils

import (
	"sort"

	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
)

// IsOptionDerivedVariant reports whether a variant has at least one selected option value.
func IsOptionDerivedVariant(v model.VariantDetailResponse) bool {
	return len(v.SelectedOptions) > 0
}

// FilterPublicVariants returns only option-derived variants (excludes internal placeholders).
func FilterPublicVariants(all []model.VariantDetailResponse) []model.VariantDetailResponse {
	if len(all) == 0 {
		return []model.VariantDetailResponse{}
	}

	public := make([]model.VariantDetailResponse, 0, len(all))
	for _, v := range all {
		if IsOptionDerivedVariant(v) {
			public = append(public, v)
		}
	}
	return public
}

// FindDefaultVariant returns the default variant for commerce fields.
// Prefers isDefault == true; otherwise the variant with the lowest ID (stable order).
func FindDefaultVariant(all []model.VariantDetailResponse) *model.VariantDetailResponse {
	if len(all) == 0 {
		return nil
	}

	sorted := make([]model.VariantDetailResponse, len(all))
	copy(sorted, all)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	for i := range sorted {
		if sorted[i].IsDefault {
			return &sorted[i]
		}
	}

	return &sorted[0]
}

// DeriveHasVariants reports whether the product is configurable (options exist and at least one public variant).
func DeriveHasVariants(productOptionsCount int, publicVariants []model.VariantDetailResponse) bool {
	return productOptionsCount > 0 && len(publicVariants) > 0
}

// DeriveAllowPurchase returns true if any variant allows purchase.
func DeriveAllowPurchase(all []model.VariantDetailResponse) bool {
	for _, v := range all {
		if v.AllowPurchase {
			return true
		}
	}
	return false
}

// DeriveIsPopular returns true if any variant is marked popular.
func DeriveIsPopular(all []model.VariantDetailResponse) bool {
	for _, v := range all {
		if v.IsPopular {
			return true
		}
	}
	return false
}

// DeriveProductPrice returns the default variant price for product-level display.
func DeriveProductPrice(all []model.VariantDetailResponse) float64 {
	if def := FindDefaultVariant(all); def != nil {
		return def.Price
	}
	return 0
}

// DerivePriceRange builds listing price range from public variants or default price for simple products.
func DerivePriceRange(
	public []model.VariantDetailResponse,
	defaultPrice float64,
	hasVariants bool,
) *model.PriceRange {
	if !hasVariants {
		if defaultPrice == 0 {
			return nil
		}
		return &model.PriceRange{Min: defaultPrice, Max: defaultPrice}
	}

	if len(public) == 0 {
		return nil
	}

	minPrice := public[0].Price
	maxPrice := public[0].Price
	for _, v := range public[1:] {
		if v.Price < minPrice {
			minPrice = v.Price
		}
		if v.Price > maxPrice {
			maxPrice = v.Price
		}
	}

	return &model.PriceRange{Min: minPrice, Max: maxPrice}
}

// ApplyAggregationSemantics sets HasVariants, TotalVariants, and price range on aggregation
// from raw repository counts (option-derived vs simple product rules).
func ApplyAggregationSemantics(agg *mapper.VariantAggregation) {
	if agg == nil {
		return
	}

	agg.HasVariants = agg.ProductOptionsCount > 0 && agg.OptionDerivedCount > 0
	agg.TotalVariants = agg.OptionDerivedCount

	if agg.HasVariants {
		return
	}

	if agg.DefaultPrice > 0 {
		agg.MinPrice = agg.DefaultPrice
		agg.MaxPrice = agg.DefaultPrice
	}
}
