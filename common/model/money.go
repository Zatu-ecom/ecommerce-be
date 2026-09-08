package model

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// Money is the shared nested money contract returned on every money-bearing
// field: major amount, exact minor units, and a formatted display string.
// It is a pure value type — constructed from cents + CurrencyInfo.
type Money struct {
	Amount      float64 `json:"amount"`
	AmountCents int64   `json:"amountCents"`
	Formatted   string  `json:"formatted"`
}

// ErrInvalidCurrency is returned when a currency fails validation.
var ErrInvalidCurrency = errors.New("invalid currency: code must be 3 letters and decimal digits 0-4")

// ErrInvalidPrecision is returned when a major amount has more fractional
// digits than the currency's DecimalDigits allows.
var ErrInvalidPrecision = errors.New("amount has more fractional digits than the currency allows")

// ErrNegativeAmount is returned when a negative major amount is provided.
var ErrNegativeAmount = errors.New("amount must not be negative")

// NewMoney builds a Money value from minor units and currency metadata.
// The major amount and formatted string are derived from cents.
func NewMoney(cents int64, ccy CurrencyInfo) Money {
	amount := FromCents(cents, ccy)
	return Money{
		Amount:      amount,
		AmountCents: cents,
		Formatted:   Format(cents, ccy),
	}
}

// Zero returns a zero Money value in the given currency.
func Zero(ccy CurrencyInfo) Money {
	return NewMoney(0, ccy)
}

// FromCents converts minor units to a major-unit float using the currency factor.
func FromCents(cents int64, ccy CurrencyInfo) float64 {
	return float64(cents) / float64(ccy.Factor())
}

// ToCents converts a major-unit amount to integer minor units.
// It validates that the amount does not exceed the currency's fractional
// precision and rounds half-up at the final digit. Excess precision is
// rejected (not silently rounded).
func (c CurrencyInfo) ToCents(amount float64) (int64, error) {
	if !c.Valid() {
		return 0, ErrInvalidCurrency
	}
	if amount < 0 {
		return 0, ErrNegativeAmount
	}
	if err := c.ValidateMajorAmount(amount); err != nil {
		return 0, err
	}
	scaled := amount * float64(c.Factor())
	// Guard against float noise at the boundary (e.g. 0.29*100 = 28.999...).
	rounded := math.Round(scaled + math.Copysign(1e-9, scaled))
	return int64(rounded), nil
}

// ValidateMajorAmount checks that the amount's fractional digits do not exceed
// the currency's DecimalDigits. It returns ErrInvalidPrecision otherwise.
func (c CurrencyInfo) ValidateMajorAmount(amount float64) error {
	if !c.Valid() {
		return ErrInvalidCurrency
	}
	if c.DecimalDigits == 0 {
		// Zero-decimal currency (e.g. JPY): reject any fractional part.
		if amount != math.Trunc(amount) {
			return ErrInvalidPrecision
		}
		return nil
	}
	s := strconv.FormatFloat(amount, 'f', -1, 64)
	if i := strings.IndexByte(s, '.'); i >= 0 {
		if frac := len(s) - i - 1; frac > c.DecimalDigits {
			return ErrInvalidPrecision
		}
	}
	return nil
}

// Format renders a minor-unit value as a display string with the currency
// symbol and exactly DecimalDigits fractional places.
func Format(cents int64, ccy CurrencyInfo) string {
	return ccy.FormatAmount(FromCents(cents, ccy))
}
