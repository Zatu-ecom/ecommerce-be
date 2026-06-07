package utils_test

import (
	"testing"

	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils"

	"github.com/stretchr/testify/assert"
)

func variant(
	id uint,
	price float64,
	isDefault, allowPurchase, isPopular bool,
	options ...model.VariantOptionResponse,
) model.VariantDetailResponse {
	return model.VariantDetailResponse{
		ID:              id,
		Price:           price,
		IsDefault:       isDefault,
		AllowPurchase:   allowPurchase,
		IsPopular:       isPopular,
		SelectedOptions: options,
	}
}

func TestIsOptionDerivedVariant(t *testing.T) {
	assert.False(t, utils.IsOptionDerivedVariant(variant(1, 10, true, true, false)))
	assert.True(t, utils.IsOptionDerivedVariant(variant(
		2, 10, false, true, false,
		model.VariantOptionResponse{OptionName: "Color", Value: "Red"},
	)))
}

func TestFilterPublicVariants(t *testing.T) {
	all := []model.VariantDetailResponse{
		variant(1, 100, true, true, false),
		variant(2, 29, false, true, false, model.VariantOptionResponse{Value: "S"}),
	}
	public := utils.FilterPublicVariants(all)
	assert.Len(t, public, 1)
	assert.Equal(t, uint(2), public[0].ID)
}

func TestFindDefaultVariant(t *testing.T) {
	t.Run("prefers isDefault", func(t *testing.T) {
		all := []model.VariantDetailResponse{
			variant(1, 10, false, true, false),
			variant(2, 20, true, true, false),
		}
		def := utils.FindDefaultVariant(all)
		assert.Equal(t, uint(2), def.ID)
	})

	t.Run("falls back to lowest id", func(t *testing.T) {
		all := []model.VariantDetailResponse{
			variant(5, 10, false, true, false),
			variant(3, 20, false, true, false),
		}
		def := utils.FindDefaultVariant(all)
		assert.Equal(t, uint(3), def.ID)
	})
}

func TestDeriveHasVariants(t *testing.T) {
	assert.False(t, utils.DeriveHasVariants(0, nil))
	assert.False(t, utils.DeriveHasVariants(2, nil))
	assert.False(t, utils.DeriveHasVariants(2, []model.VariantDetailResponse{}))
	assert.True(t, utils.DeriveHasVariants(1, []model.VariantDetailResponse{
		variant(1, 10, false, true, false, model.VariantOptionResponse{}),
	}))
}

func TestDeriveAllowPurchaseAndIsPopular(t *testing.T) {
	all := []model.VariantDetailResponse{
		variant(1, 10, true, false, false),
		variant(2, 20, false, true, false),
	}
	assert.True(t, utils.DeriveAllowPurchase(all))
	assert.False(t, utils.DeriveIsPopular(all))

	all[1].IsPopular = true
	assert.True(t, utils.DeriveIsPopular(all))
}

func TestDeriveProductPrice(t *testing.T) {
	all := []model.VariantDetailResponse{
		variant(1, 100, true, true, false),
		variant(2, 50, false, true, false, model.VariantOptionResponse{Value: "M"}),
	}
	assert.Equal(t, 100.0, utils.DeriveProductPrice(all))
}

func TestDerivePriceRange(t *testing.T) {
	t.Run("simple product min equals max", func(t *testing.T) {
		pr := utils.DerivePriceRange(nil, 399.99, false)
		assert.Equal(t, 399.99, pr.Min)
		assert.Equal(t, 399.99, pr.Max)
	})

	t.Run("configurable min max over public", func(t *testing.T) {
		public := []model.VariantDetailResponse{
			variant(1, 29.99, true, true, false, model.VariantOptionResponse{}),
			variant(2, 34.99, false, true, false, model.VariantOptionResponse{}),
		}
		pr := utils.DerivePriceRange(public, 29.99, true)
		assert.Equal(t, 29.99, pr.Min)
		assert.Equal(t, 34.99, pr.Max)
	})
}

func TestApplyAggregationSemantics(t *testing.T) {
	t.Run("simple product", func(t *testing.T) {
		agg := &mapper.VariantAggregation{
			ProductOptionsCount: 0,
			OptionDerivedCount:  0,
			DefaultPrice:        100,
			MinPrice:            50,
			MaxPrice:            200,
		}
		utils.ApplyAggregationSemantics(agg)
		assert.False(t, agg.HasVariants)
		assert.Equal(t, 0, agg.TotalVariants)
		assert.Equal(t, 100.0, agg.MinPrice)
		assert.Equal(t, 100.0, agg.MaxPrice)
	})

	t.Run("configurable product keeps option-derived prices", func(t *testing.T) {
		agg := &mapper.VariantAggregation{
			ProductOptionsCount: 2,
			OptionDerivedCount:  3,
			DefaultPrice:        29.99,
			MinPrice:            29.99,
			MaxPrice:            34.99,
		}
		utils.ApplyAggregationSemantics(agg)
		assert.True(t, agg.HasVariants)
		assert.Equal(t, 3, agg.TotalVariants)
		assert.Equal(t, 29.99, agg.MinPrice)
		assert.Equal(t, 34.99, agg.MaxPrice)
	})
}
