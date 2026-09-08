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

// DeriveProductPriceCents returns the default variant price (cents) for product-level display.
func DeriveProductPriceCents(all []model.VariantDetailResponse) int64 {
	if def := FindDefaultVariant(all); def != nil {
		return def.Price.AmountCents
	}
	return 0
}

// DerivePriceRangeCents builds listing price range (cents) from public variants or default price for simple products.
func DerivePriceRangeCents(
	public []model.VariantDetailResponse,
	defaultPriceCents int64,
	hasVariants bool,
) *model.PriceRangeCents {
	if !hasVariants {
		if defaultPriceCents == 0 {
			return nil
		}
		return &model.PriceRangeCents{MinCents: defaultPriceCents, MaxCents: defaultPriceCents}
	}

	if len(public) == 0 {
		return nil
	}

	minPrice := public[0].Price.AmountCents
	maxPrice := public[0].Price.AmountCents
	for _, v := range public[1:] {
		if v.Price.AmountCents < minPrice {
			minPrice = v.Price.AmountCents
		}
		if v.Price.AmountCents > maxPrice {
			maxPrice = v.Price.AmountCents
		}
	}

	return &model.PriceRangeCents{MinCents: minPrice, MaxCents: maxPrice}
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

	if agg.DefaultPriceCents > 0 {
		agg.MinPriceCents = agg.DefaultPriceCents
		agg.MaxPriceCents = agg.DefaultPriceCents
	}
}
