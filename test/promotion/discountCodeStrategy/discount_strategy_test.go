package discountCodeStrategy_test

import (
	"context"
	"testing"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/service/discountCodeStrategy"

	"github.com/stretchr/testify/require"
)

func TestGetDiscountStrategy(t *testing.T) {
	require.NotNil(t, discountCodeStrategy.GetDiscountStrategy(entity.DiscountPercentage))
	require.NotNil(t, discountCodeStrategy.GetDiscountStrategy(entity.DiscountFixedAmount))
	require.NotNil(t, discountCodeStrategy.GetDiscountStrategy(entity.DiscountFreeShipping))
	require.NotNil(t, discountCodeStrategy.GetDiscountStrategy(entity.DiscountBuyXGetY))
	require.Nil(t, discountCodeStrategy.GetDiscountStrategy(entity.DiscountType("unknown")))
}

func TestPercentageDiscountStrategy(t *testing.T) {
	ctx := context.Background()
	strategy := &discountCodeStrategy.PercentageDiscountStrategy{}
	maxCap := int64(500)
	code := &entity.DiscountCode{Value: 10, MaxDiscountAmountCents: &maxCap}
	req := &model.CouponCartRequest{
		Items: []model.CartItem{
			{ItemID: "1", TotalCents: 10000, Quantity: 1, PriceCents: 10000},
		},
	}

	merch, ship, err := strategy.Calculate(ctx, code, req, []string{"1"})
	require.NoError(t, err)
	require.Equal(t, int64(500), merch)
	require.Equal(t, int64(0), ship)

	code.MaxDiscountAmountCents = nil
	merch, _, err = strategy.Calculate(ctx, code, req, []string{"1"})
	require.NoError(t, err)
	require.Equal(t, int64(1000), merch)

	merch, _, err = strategy.Calculate(ctx, code, req, []string{"missing"})
	require.NoError(t, err)
	require.Equal(t, int64(0), merch)

	// Zero / negative value → no discount
	code.Value = 0
	merch, _, err = strategy.Calculate(ctx, code, req, []string{"1"})
	require.NoError(t, err)
	require.Equal(t, int64(0), merch)

	code.Value = -5
	merch, _, err = strategy.Calculate(ctx, code, req, []string{"1"})
	require.NoError(t, err)
	require.Equal(t, int64(0), merch)

	// Cap above computed discount is a no-op
	code.Value = 10
	highCap := int64(5000)
	code.MaxDiscountAmountCents = &highCap
	merch, _, err = strategy.Calculate(ctx, code, req, []string{"1"})
	require.NoError(t, err)
	require.Equal(t, int64(1000), merch)
}

func TestFixedAmountDiscountStrategy(t *testing.T) {
	ctx := context.Background()
	strategy := &discountCodeStrategy.FixedAmountDiscountStrategy{}
	code := &entity.DiscountCode{Value: 2500}
	req := &model.CouponCartRequest{
		Items: []model.CartItem{
			{ItemID: "1", TotalCents: 1000, Quantity: 1, PriceCents: 1000},
		},
	}

	merch, ship, err := strategy.Calculate(ctx, code, req, []string{"1"})
	require.NoError(t, err)
	require.Equal(t, int64(1000), merch)
	require.Equal(t, int64(0), ship)

	code.Value = 500
	req.Items[0].TotalCents = 5000
	merch, _, err = strategy.Calculate(ctx, code, req, []string{"1"})
	require.NoError(t, err)
	require.Equal(t, int64(500), merch)

	// Empty eligible set
	merch, _, err = strategy.Calculate(ctx, code, req, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), merch)

	code.Value = 0
	req.Items[0].TotalCents = 5000
	merch, _, err = strategy.Calculate(ctx, code, req, []string{"1"})
	require.NoError(t, err)
	require.Equal(t, int64(0), merch)
}

func TestFreeShippingDiscountStrategy(t *testing.T) {
	ctx := context.Background()
	strategy := &discountCodeStrategy.FreeShippingDiscountStrategy{}
	code := &entity.DiscountCode{}
	req := &model.CouponCartRequest{ShippingCents: 7500}

	merch, ship, err := strategy.Calculate(ctx, code, req, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), merch)
	require.Equal(t, int64(7500), ship)

	req.ShippingCents = 0
	merch, ship, err = strategy.Calculate(ctx, code, req, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), merch)
	require.Equal(t, int64(0), ship)
}

func TestBuyXGetYDiscountStrategy(t *testing.T) {
	ctx := context.Background()
	strategy := &discountCodeStrategy.BuyXGetYDiscountStrategy{}
	code := &entity.DiscountCode{
		Metadata: db.JSONMap{
			"buyQuantity":        1,
			"getQuantity":        1,
			"getDiscountPercent": 100,
		},
	}
	req := &model.CouponCartRequest{
		Items: []model.CartItem{
			{ItemID: "1", Quantity: 2, PriceCents: 99900, TotalCents: 199800},
		},
	}

	merch, ship, err := strategy.Calculate(ctx, code, req, []string{"1"})
	require.NoError(t, err)
	require.Equal(t, int64(99900), merch)
	require.Equal(t, int64(0), ship)

	// Not enough units
	req.Items[0].Quantity = 1
	req.Items[0].TotalCents = 99900
	merch, _, err = strategy.Calculate(ctx, code, req, []string{"1"})
	require.NoError(t, err)
	require.Equal(t, int64(0), merch)

	// Partial get percent
	req.Items[0].Quantity = 2
	req.Items[0].TotalCents = 199800
	code.Metadata["getDiscountPercent"] = 50
	merch, _, err = strategy.Calculate(ctx, code, req, []string{"1"})
	require.NoError(t, err)
	require.Equal(t, int64(49950), merch)

	// Cheapest units discounted across mixed prices
	v2 := uint(2)
	req.Items = []model.CartItem{
		{ItemID: "a", Quantity: 1, PriceCents: 10000, TotalCents: 10000, VariantID: &v2},
		{ItemID: "b", Quantity: 1, PriceCents: 30000, TotalCents: 30000},
	}
	code.Metadata = db.JSONMap{
		"buyQuantity":        1,
		"getQuantity":        1,
		"getDiscountPercent": 100,
	}
	merch, _, err = strategy.Calculate(ctx, code, req, []string{"a", "b"})
	require.NoError(t, err)
	require.Equal(t, int64(10000), merch)

	// Invalid metadata
	code.Metadata = db.JSONMap{"buyQuantity": 0}
	merch, _, err = strategy.Calculate(ctx, code, req, []string{"a", "b"})
	require.NoError(t, err)
	require.Equal(t, int64(0), merch)

	code.Metadata = db.JSONMap{
		"buyQuantity":        1,
		"getQuantity":        1,
		"getDiscountPercent": 150,
	}
	merch, _, err = strategy.Calculate(ctx, code, req, []string{"a", "b"})
	require.NoError(t, err)
	require.Equal(t, int64(0), merch)
}
