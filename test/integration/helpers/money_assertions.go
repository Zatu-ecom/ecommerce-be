package helpers

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertMoney asserts a nested Money object shape: amount/amountCents/formatted.
// Shared across the money standardization suites (US2/US4/US5).
func AssertMoney(t require.TestingT, node any) {
	m, ok := node.(map[string]any)
	require.True(t, ok, "money node should be an object")
	_, hasAmount := m["amount"]
	_, hasCents := m["amountCents"]
	_, hasFormatted := m["formatted"]
	assert.True(t, hasAmount, "money.amount present")
	assert.True(t, hasCents, "money.amountCents present")
	assert.True(t, hasFormatted, "money.formatted present")
}

// AssertMoneyCents asserts the minor-unit (amountCents) value of a Money node.
func AssertMoneyCents(t require.TestingT, node any, want int64) {
	m, ok := node.(map[string]any)
	require.True(t, ok, "money node should be an object")
	cents, ok := m["amountCents"].(float64)
	require.True(t, ok, "amountCents should be a number")
	assert.Equal(t, float64(want), cents)
}

// AssertMoneyAmount asserts the major-unit (amount) value of a Money node.
func AssertMoneyAmount(t require.TestingT, node any, want float64) {
	m, ok := node.(map[string]any)
	require.True(t, ok, "money node should be an object")
	amt, ok := m["amount"].(float64)
	require.True(t, ok, "amount should be a number")
	assert.InDelta(t, want, amt, 0.0001)
}

// AssertCurrency asserts a parent currency object shape (code/symbol/decimalDigits).
func AssertCurrency(t require.TestingT, node any, wantCode string, wantDigits int) {
	m, ok := node.(map[string]any)
	require.True(t, ok, "currency node should be an object")
	assert.Equal(t, wantCode, m["code"])
	assert.Equal(t, float64(wantDigits), m["decimalDigits"])
}

// MoneyAmount extracts the major-unit amount from a nested Money node.
func MoneyAmount(node any) float64 {
	m, ok := node.(map[string]any)
	if !ok {
		return 0
	}
	amt, _ := m["amount"].(float64)
	return amt
}

// MoneyCents extracts the minor-unit value from a nested Money node.
func MoneyCents(node any) int64 {
	m, ok := node.(map[string]any)
	if !ok {
		return 0
	}
	cents, _ := m["amountCents"].(float64)
	return int64(cents)
}
