package model_test

import (
	"testing"

	commonModel "ecommerce-be/common/model"

	"github.com/stretchr/testify/assert"
)

// ─── Factor for digits 0–4 ──────────────────────────────────────────────────
func TestCurrencyFactor(t *testing.T) {
	cases := []struct {
		digits int
		want   int64
	}{
		{0, 1},
		{1, 10},
		{2, 100},
		{3, 1000},
		{4, 10000},
	}
	for _, tc := range cases {
		ccy := commonModel.CurrencyInfo{Code: "XXX", Symbol: "S", DecimalDigits: tc.digits}
		assert.Equal(t, tc.want, ccy.Factor(), "digits %d", tc.digits)
	}
}

// ─── Valid currency rules ───────────────────────────────────────────────────
func TestCurrencyValid(t *testing.T) {
	assert.True(t, (commonModel.CurrencyInfo{Code: "INR", Symbol: "₹", DecimalDigits: 2}).Valid())
	assert.True(t, (commonModel.CurrencyInfo{Code: "JPY", Symbol: "¥", DecimalDigits: 0}).Valid())
	assert.False(t, (commonModel.CurrencyInfo{Code: "IN", Symbol: "₹", DecimalDigits: 2}).Valid())
	assert.False(t, (commonModel.CurrencyInfo{Code: "ABCD", Symbol: "S", DecimalDigits: 2}).Valid())
	assert.False(t, (commonModel.CurrencyInfo{Code: "XXX", Symbol: "S", DecimalDigits: 5}).Valid())
	assert.False(t, (commonModel.CurrencyInfo{Code: "XXX", Symbol: "S", DecimalDigits: -1}).Valid())
}

// ─── FormatAmount with symbol and digit count ───────────────────────────────
func TestFormatAmount(t *testing.T) {
	inr := commonModel.CurrencyInfo{Code: "INR", Symbol: "₹", DecimalDigits: 2}
	assert.Equal(t, "₹99.99", inr.FormatAmount(99.99))
	assert.Equal(t, "₹0.00", inr.FormatAmount(0))

	jpy := commonModel.CurrencyInfo{Code: "JPY", Symbol: "¥", DecimalDigits: 0}
	assert.Equal(t, "¥100", jpy.FormatAmount(100))
}
