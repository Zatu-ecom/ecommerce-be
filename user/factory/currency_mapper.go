package factory

import (
	commonModel "ecommerce-be/common/model"
	"ecommerce-be/user/model"
)

// ToCurrencyInfo maps a user-module CurrencyResponse to the shared
// common/model.CurrencyInfo contract. The mapper lives in the user module
// (which owns currency/settings data); common/model must never import user.
func ToCurrencyInfo(c *model.CurrencyResponse) commonModel.CurrencyInfo {
	if c == nil {
		return commonModel.CurrencyInfo{}
	}
	return commonModel.CurrencyInfo{
		Code:          c.Code,
		Symbol:        c.Symbol,
		DecimalDigits: c.DecimalDigits,
	}
}
