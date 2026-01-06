package factory

import (
	"time"

	"ecommerce-be/user/entity"
	"ecommerce-be/user/model"
)

// ========================================
// COUNTRY RESPONSE BUILDERS
// ========================================

// BuildCountryResponse converts a country entity to a response model
func BuildCountryResponse(country *entity.Country) model.CountryResponse {
	return model.CountryResponse{
		ID: country.ID,
		CountryBase: model.CountryBase{
			Code:       country.Code,
			CodeAlpha3: country.CodeAlpha3,
			Name:       country.Name,
			NativeName: country.NativeName,
			PhoneCode:  country.PhoneCode,
			Region:     country.Region,
			FlagEmoji:  country.FlagEmoji,
		},
		IsActive: country.IsActive,
	}
}

// BuildCountryResponsePtr converts a country entity to a response model pointer
func BuildCountryResponsePtr(country *entity.Country) *model.CountryResponse {
	if country == nil || country.ID == 0 {
		return nil
	}
	response := BuildCountryResponse(country)
	return &response
}

// BuildCountryDetailResponse converts a country entity to a detailed response with currencies
func BuildCountryDetailResponse(country *entity.Country) model.CountryDetailResponse {
	response := model.CountryDetailResponse{
		CountryResponse: BuildCountryResponse(country),
		CreatedAt:       country.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       country.UpdatedAt.Format(time.RFC3339),
	}

	// Build currency list if available
	if len(country.Currencies) > 0 {
		currencies := make([]model.CurrencyInCountryResponse, 0, len(country.Currencies))
		for _, currency := range country.Currencies {
			currencies = append(currencies, model.CurrencyInCountryResponse{
				ID:        currency.ID,
				Code:      currency.Code,
				Name:      currency.Name,
				Symbol:    currency.Symbol,
				IsPrimary: false, // TODO: Get from country_currency junction table
			})
		}
		response.Currencies = currencies
	}

	return response
}

// BuildCountryListResponse converts a list of country entities to response models
func BuildCountryListResponse(countries []entity.Country) []model.CountryResponse {
	responses := make([]model.CountryResponse, 0, len(countries))
	for _, country := range countries {
		responses = append(responses, BuildCountryResponse(&country))
	}
	return responses
}

// BuildCountryDetailListResponse converts a list of country entities to detailed responses
func BuildCountryDetailListResponse(countries []entity.Country) []model.CountryDetailResponse {
	responses := make([]model.CountryDetailResponse, 0, len(countries))
	for _, country := range countries {
		responses = append(responses, BuildCountryDetailResponse(&country))
	}
	return responses
}
