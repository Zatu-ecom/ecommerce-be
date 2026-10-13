package model

import "fmt"

// CurrencyInfo is the shared currency metadata contract returned on money-bearing
// parent resources. It is a pure value type — it must not look up sellers,
// settings, or the database.
type CurrencyInfo struct {
	Code          string `json:"code"`
	Symbol        string `json:"symbol"`
	DecimalDigits int    `json:"decimalDigits"`
}

// Factor returns the minor-unit multiplier for this currency (10^DecimalDigits).
// Examples: 2 digits → 100, 0 digits → 1, 3 digits → 1000.
func (c CurrencyInfo) Factor() int64 {
	factor := int64(1)
	for i := 0; i < c.DecimalDigits; i++ {
		factor *= 10
	}
	return factor
}

// Valid returns whether the currency has a valid ISO-style code and a
// supported decimal digit count (0–4).
func (c CurrencyInfo) Valid() bool {
	return len(c.Code) == 3 && c.DecimalDigits >= 0 && c.DecimalDigits <= 4
}

// FormatAmount renders a major-unit amount with the currency symbol and exactly
// DecimalDigits fractional places. The amount is formatted without grouping.
func (c CurrencyInfo) FormatAmount(amount float64) string {
	return fmt.Sprintf("%s%.*f", c.Symbol, c.DecimalDigits, amount)
}
