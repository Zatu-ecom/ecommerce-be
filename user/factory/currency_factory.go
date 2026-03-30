package factory

import (
	"time"

	"ecommerce-be/user/entity"
	"ecommerce-be/user/model"
)

// ========================================
// CURRENCY RESPONSE BUILDERS
// ========================================

// BuildCurrencyResponse converts a currency entity to a response model
func BuildCurrencyResponse(currency *entity.Currency) model.CurrencyResponse {
	return model.CurrencyResponse{
		ID: currency.ID,
		CurrencyBase: model.CurrencyBase{
			Code:          currency.Code,
			Name:          currency.Name,
			Symbol:        currency.Symbol,
			SymbolNative:  currency.SymbolNative,
			DecimalDigits: currency.DecimalDigits,
		},
		IsActive: currency.IsActive,
	}
}

// BuildCurrencyDetailResponse converts a currency entity to a detailed response with countries
func BuildCurrencyDetailResponse(currency *entity.Currency) model.CurrencyDetailResponse {
	response := model.CurrencyDetailResponse{
		CurrencyResponse: BuildCurrencyResponse(currency),
		CreatedAt:        currency.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        currency.UpdatedAt.Format(time.RFC3339),
	}

	// Build country list if available
	if len(currency.Countries) > 0 {
		countries := make([]model.CountryInCurrencyResponse, 0, len(currency.Countries))
		for _, country := range currency.Countries {
			countries = append(countries, model.CountryInCurrencyResponse{
				ID:        country.ID,
				Code:      country.Code,
				Name:      country.Name,
				FlagEmoji: country.FlagEmoji,
				IsPrimary: false, // TODO: Get from country_currency junction table
			})
		}
		response.Countries = countries
	}

	return response
}

// BuildCurrencyListResponse converts a list of currency entities to response models
func BuildCurrencyListResponse(currencies []entity.Currency) []model.CurrencyResponse {
	responses := make([]model.CurrencyResponse, 0, len(currencies))
	for _, currency := range currencies {
		responses = append(responses, BuildCurrencyResponse(&currency))
	}
	return responses
}

// BuildCurrencyDetailListResponse converts a list of currency entities to detailed responses
func BuildCurrencyDetailListResponse(currencies []entity.Currency) []model.CurrencyDetailResponse {
	responses := make([]model.CurrencyDetailResponse, 0, len(currencies))
	for _, currency := range currencies {
		responses = append(responses, BuildCurrencyDetailResponse(&currency))
	}
	return responses
}
