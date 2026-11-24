package model_test

import (
	"testing"

	commonModel "ecommerce-be/common/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── ToCents factor table (digits 0–4) ──────────────────────────────────────
func TestToCentsFactorTable(t *testing.T) {
	cases := []struct {
		name       string
		digits     int
		amount     float64
		wantCents  int64
	}{
		{"INR 2dp", 2, 99.99, 9999},
		{"USD 2dp", 2, 0.01, 1},
		{"JPY 0dp", 0, 100, 100},
		{"JPY 0dp single", 0, 5, 5},
		{"KWD 3dp", 3, 1.250, 1250},
		{"4dp", 4, 1.2345, 12345},
		{"1dp", 1, 12.3, 123},
		{"zero", 2, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ccy := commonModel.CurrencyInfo{Code: "XXX", Symbol: "S", DecimalDigits: tc.digits}
			got, err := ccy.ToCents(tc.amount)
			require.NoError(t, err)
			assert.Equal(t, tc.wantCents, got)
		})
	}
}

// ─── Half-up rounding at the final digit ────────────────────────────────────
func TestToCentsHalfUpRounding(t *testing.T) {
	ccy := commonModel.CurrencyInfo{Code: "INR", Symbol: "₹", DecimalDigits: 2}

	// Exact boundary values scale cleanly with the float-noise guard.
	cases := []struct {
		amount    float64
		wantCents int64
	}{
		{0.29, 29},
		{0.30, 30},
		{1.00, 100},
		{2.50, 250},
		{1299.50, 129950},
	}
	for _, tc := range cases {
		got, err := ccy.ToCents(tc.amount)
		require.NoError(t, err)
		assert.Equal(t, tc.wantCents, got, "amount %v", tc.amount)
	}
}

// ─── Excess-precision rejection ─────────────────────────────────────────────
func TestToCentsRejectsExcessPrecision(t *testing.T) {
	inr := commonModel.CurrencyInfo{Code: "INR", Symbol: "₹", DecimalDigits: 2}
	jpy := commonModel.CurrencyInfo{Code: "JPY", Symbol: "¥", DecimalDigits: 0}
	kwd := commonModel.CurrencyInfo{Code: "KWD", Symbol: "د.ك", DecimalDigits: 3}

	cases := []struct {
		name   string
		ccy    commonModel.CurrencyInfo
		amount float64
	}{
		{"INR 3dp", inr, 99.999},
		{"JPY fractional", jpy, 100.5},
		{"JPY 2dp", jpy, 1.50},
		{"KWD 4dp", kwd, 1.2345},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.ccy.ToCents(tc.amount)
			assert.ErrorIs(t, err, commonModel.ErrInvalidPrecision)
		})
	}
}

// ─── Negative and invalid currency rejection ────────────────────────────────
func TestToCentsRejectsNegativeAndInvalid(t *testing.T) {
	inr := commonModel.CurrencyInfo{Code: "INR", Symbol: "₹", DecimalDigits: 2}
	_, err := inr.ToCents(-1)
	assert.ErrorIs(t, err, commonModel.ErrNegativeAmount)

	bad := commonModel.CurrencyInfo{Code: "IN", Symbol: "₹", DecimalDigits: 2}
	_, err = bad.ToCents(5)
	assert.ErrorIs(t, err, commonModel.ErrInvalidCurrency)

	badDigits := commonModel.CurrencyInfo{Code: "XXX", Symbol: "S", DecimalDigits: 5}
	_, err = badDigits.ToCents(5)
	assert.ErrorIs(t, err, commonModel.ErrInvalidCurrency)
}

// ─── FromCents / NewMoney dual shape ────────────────────────────────────────
func TestNewMoneyDerivesAmountAndFormatted(t *testing.T) {
	inr := commonModel.CurrencyInfo{Code: "INR", Symbol: "₹", DecimalDigits: 2}
	m := commonModel.NewMoney(9999, inr)
	assert.Equal(t, int64(9999), m.AmountCents)
	assert.InDelta(t, 99.99, m.Amount, 0.0001)
	assert.Equal(t, "₹99.99", m.Formatted)

	jpy := commonModel.CurrencyInfo{Code: "JPY", Symbol: "¥", DecimalDigits: 0}
	m = commonModel.NewMoney(100, jpy)
	assert.Equal(t, int64(100), m.AmountCents)
	assert.InDelta(t, 100, m.Amount, 0.0001)
	assert.Equal(t, "¥100", m.Formatted)
}

// ─── Zero and large values ──────────────────────────────────────────────────
func TestZeroAndLargeValues(t *testing.T) {
	inr := commonModel.CurrencyInfo{Code: "INR", Symbol: "₹", DecimalDigits: 2}
	z := commonModel.Zero(inr)
	assert.Equal(t, int64(0), z.AmountCents)
	assert.InDelta(t, 0, z.Amount, 0.0001)
	assert.Equal(t, "₹0.00", z.Formatted)

	// Large values stay within int64 range.
	big, err := inr.ToCents(99999999999.99)
	require.NoError(t, err)
	assert.Equal(t, int64(9999999999999), big)
}
