package utils_test

import (
	"testing"

	commonModel "ecommerce-be/common/model"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils"

	"github.com/stretchr/testify/assert"
)

// inr is a 2-decimal currency used across these tests; cents values are
// derived via commonModel.NewMoney so the Money contract is exercised too.
var inr = commonModel.CurrencyInfo{Code: "INR", Symbol: "₹", DecimalDigits: 2}

// variant builds a VariantDetailResponse with a Money price from cents.
func variant(
	id uint,
	priceCents int64,
	isDefault, allowPurchase, isPopular bool,
	options ...model.VariantOptionResponse,
) model.VariantDetailResponse {
	return model.VariantDetailResponse{
		ID:              id,
		Price:           commonModel.NewMoney(priceCents, inr),
		IsDefault:       isDefault,
		AllowPurchase:   allowPurchase,
		IsPopular:       isPopular,
		SelectedOptions: options,
	}
}

func TestIsOptionDerivedVariant(t *testing.T) {
	assert.False(t, utils.IsOptionDerivedVariant(variant(1, 1000, true, true, false)))
	assert.True(t, utils.IsOptionDerivedVariant(variant(
		2, 1000, false, true, false,
		model.VariantOptionResponse{OptionName: "Color", Value: "Red"},
	)))
}

func TestFilterPublicVariants(t *testing.T) {
	all := []model.VariantDetailResponse{
		variant(1, 10000, true, true, false),
		variant(2, 2900, false, true, false, model.VariantOptionResponse{Value: "S"}),
	}
	public := utils.FilterPublicVariants(all)
	assert.Len(t, public, 1)
	assert.Equal(t, uint(2), public[0].ID)
}

func TestFindDefaultVariant(t *testing.T) {
	t.Run("prefers isDefault", func(t *testing.T) {
		all := []model.VariantDetailResponse{
			variant(1, 1000, false, true, false),
			variant(2, 2000, true, true, false),
		}
		def := utils.FindDefaultVariant(all)
		assert.Equal(t, uint(2), def.ID)
	})

	t.Run("falls back to lowest id", func(t *testing.T) {
		all := []model.VariantDetailResponse{
			variant(5, 1000, false, true, false),
			variant(3, 2000, false, true, false),
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
		variant(1, 1000, false, true, false, model.VariantOptionResponse{}),
	}))
}

func TestDeriveAllowPurchaseAndIsPopular(t *testing.T) {
	all := []model.VariantDetailResponse{
		variant(1, 1000, true, false, false),
		variant(2, 2000, false, true, false),
	}
	assert.True(t, utils.DeriveAllowPurchase(all))
	assert.False(t, utils.DeriveIsPopular(all))

	all[1].IsPopular = true
	assert.True(t, utils.DeriveIsPopular(all))
}

func TestDeriveProductPriceCents(t *testing.T) {
	all := []model.VariantDetailResponse{
		variant(1, 10000, true, true, false),
		variant(2, 5000, false, true, false, model.VariantOptionResponse{Value: "M"}),
	}
	assert.Equal(t, int64(10000), utils.DeriveProductPriceCents(all))
}

func TestDerivePriceRangeCents(t *testing.T) {
	t.Run("simple product min equals max", func(t *testing.T) {
		pr := utils.DerivePriceRangeCents(nil, 39999, false)
		if assert.NotNil(t, pr) {
			assert.Equal(t, int64(39999), pr.MinCents)
			assert.Equal(t, int64(39999), pr.MaxCents)
		}
	})

	t.Run("simple product zero price yields nil", func(t *testing.T) {
		assert.Nil(t, utils.DerivePriceRangeCents(nil, 0, false))
	})

	t.Run("configurable min max over public", func(t *testing.T) {
		public := []model.VariantDetailResponse{
			variant(1, 2999, true, true, false, model.VariantOptionResponse{}),
			variant(2, 3499, false, true, false, model.VariantOptionResponse{}),
		}
		pr := utils.DerivePriceRangeCents(public, 2999, true)
		if assert.NotNil(t, pr) {
			assert.Equal(t, int64(2999), pr.MinCents)
			assert.Equal(t, int64(3499), pr.MaxCents)
		}
	})
}

func TestApplyAggregationSemantics(t *testing.T) {
	t.Run("simple product", func(t *testing.T) {
		agg := &mapper.VariantAggregation{
			ProductOptionsCount: 0,
			OptionDerivedCount:  0,
			DefaultPriceCents:   10000,
			MinPriceCents:       5000,
			MaxPriceCents:       20000,
		}
		utils.ApplyAggregationSemantics(agg)
		assert.False(t, agg.HasVariants)
		assert.Equal(t, 0, agg.TotalVariants)
		assert.Equal(t, int64(10000), agg.MinPriceCents)
		assert.Equal(t, int64(10000), agg.MaxPriceCents)
	})

	t.Run("configurable product keeps option-derived prices", func(t *testing.T) {
		agg := &mapper.VariantAggregation{
			ProductOptionsCount: 2,
			OptionDerivedCount:  3,
			DefaultPriceCents:   2999,
			MinPriceCents:       2999,
			MaxPriceCents:       3499,
		}
		utils.ApplyAggregationSemantics(agg)
		assert.True(t, agg.HasVariants)
		assert.Equal(t, 3, agg.TotalVariants)
		assert.Equal(t, int64(2999), agg.MinPriceCents)
		assert.Equal(t, int64(3499), agg.MaxPriceCents)
	})
}
