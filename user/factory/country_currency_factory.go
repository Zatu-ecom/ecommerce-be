package factory

import (
	"time"

	"ecommerce-be/user/entity"
	"ecommerce-be/user/model"
)

// ========================================
// COUNTRY-CURRENCY RESPONSE BUILDERS
// ========================================

// BuildCountryCurrencySimpleResponse converts a country-currency entity to a simple response
func BuildCountryCurrencySimpleResponse(mapping *entity.CountryCurrency) *model.CountryCurrencySimpleResponse {
	return &model.CountryCurrencySimpleResponse{
		ID:         mapping.ID,
		CountryID:  mapping.CountryID,
		CurrencyID: mapping.CurrencyID,
		IsPrimary:  mapping.IsPrimary,
		CreatedAt:  mapping.CreatedAt.Format(time.RFC3339),
	}
}

// BuildCountryCurrencyListResponse builds a list response with currencies for a country
func BuildCountryCurrencyListResponse(
	countryID uint,
	mappings []entity.CountryCurrency,
) *model.CountryCurrencyListResponse {
	currencies := make([]model.CurrencyWithPrimaryResponse, 0, len(mappings))

	for _, mapping := range mappings {
		currencies = append(currencies, model.CurrencyWithPrimaryResponse{
			CurrencyResponse: BuildCurrencyResponse(&mapping.Currency),
			MappingID:        mapping.ID,
			IsPrimary:        mapping.IsPrimary,
		})
	}

	return &model.CountryCurrencyListResponse{
		CountryID:  countryID,
		Currencies: currencies,
	}
}
