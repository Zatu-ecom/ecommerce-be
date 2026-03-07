package service

import (
	"context"

	"ecommerce-be/user/entity"
	userErrors "ecommerce-be/user/error"
	"ecommerce-be/user/factory"
	"ecommerce-be/user/model"
	"ecommerce-be/user/repository"
)

// CountryCurrencyService defines the interface for country-currency business logic
type CountryCurrencyService interface {
	// Query methods
	GetCurrenciesByCountryID(
		ctx context.Context,
		countryID uint,
	) (*model.CountryCurrencyListResponse, error)

	// Mutation methods
	AddCurrencyToCountry(
		ctx context.Context,
		countryID uint,
		req model.CountryCurrencyCreateRequest,
	) (*model.CountryCurrencySimpleResponse, error)
	BulkAddCurrenciesToCountry(
		ctx context.Context,
		countryID uint,
		req model.CountryCurrencyBulkRequest,
	) ([]model.CountryCurrencySimpleResponse, error)
	UpdateCountryCurrency(
		ctx context.Context,
		countryID, currencyID uint,
		req model.CountryCurrencyUpdateRequest,
	) (*model.CountryCurrencySimpleResponse, error)
	RemoveCurrencyFromCountry(ctx context.Context, countryID, currencyID uint) error
}

// CountryCurrencyServiceImpl implements the CountryCurrencyService interface
type CountryCurrencyServiceImpl struct {
	countryCurrencyRepo repository.CountryCurrencyRepository
	countryRepo         repository.CountryRepository
	currencyRepo        repository.CurrencyRepository
}

// NewCountryCurrencyService creates a new instance of CountryCurrencyService
func NewCountryCurrencyService(
	countryCurrencyRepo repository.CountryCurrencyRepository,
	countryRepo repository.CountryRepository,
	currencyRepo repository.CurrencyRepository,
) *CountryCurrencyServiceImpl {
	return &CountryCurrencyServiceImpl{
		countryCurrencyRepo: countryCurrencyRepo,
		countryRepo:         countryRepo,
		currencyRepo:        currencyRepo,
	}
}

// GetCurrenciesByCountryID retrieves all currencies mapped to a country
func (s *CountryCurrencyServiceImpl) GetCurrenciesByCountryID(
	ctx context.Context,
	countryID uint,
) (*model.CountryCurrencyListResponse, error) {
	// Verify country exists
	_, err := s.countryRepo.FindByID(ctx, countryID)
	if err != nil {
		return nil, err
	}

	// Get all currency mappings for this country
	mappings, err := s.countryCurrencyRepo.FindByCountryID(ctx, countryID)
	if err != nil {
		return nil, err
	}

	// Build response
	response := factory.BuildCountryCurrencyListResponse(countryID, mappings)
	return response, nil
}

// AddCurrencyToCountry adds a currency to a country
func (s *CountryCurrencyServiceImpl) AddCurrencyToCountry(
	ctx context.Context,
	countryID uint,
	req model.CountryCurrencyCreateRequest,
) (*model.CountryCurrencySimpleResponse, error) {
	// Verify country exists
	_, err := s.countryRepo.FindByID(ctx, countryID)
	if err != nil {
		return nil, err
	}

	// Verify currency exists
	_, err = s.currencyRepo.FindByID(ctx, req.CurrencyID)
	if err != nil {
		return nil, err
	}

	// Check if mapping already exists
	existing, err := s.countryCurrencyRepo.FindByCountryAndCurrency(ctx, countryID, req.CurrencyID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, userErrors.ErrCountryCurrencyExists
	}

	// Create mapping
	mapping := &entity.CountryCurrency{
		CountryID:  countryID,
		CurrencyID: req.CurrencyID,
		IsPrimary:  req.IsPrimary,
	}

	if err := s.countryCurrencyRepo.Create(ctx, mapping); err != nil {
		return nil, err
	}

	// Build response
	response := factory.BuildCountryCurrencySimpleResponse(mapping)
	return response, nil
}

// BulkAddCurrenciesToCountry adds multiple currencies to a country
func (s *CountryCurrencyServiceImpl) BulkAddCurrenciesToCountry(
	ctx context.Context,
	countryID uint,
	req model.CountryCurrencyBulkRequest,
) ([]model.CountryCurrencySimpleResponse, error) {
	// Verify country exists
	_, err := s.countryRepo.FindByID(ctx, countryID)
	if err != nil {
		return nil, err
	}

	var responses []model.CountryCurrencySimpleResponse

	for _, item := range req.Currencies {
		// Verify currency exists
		_, err := s.currencyRepo.FindByID(ctx, item.CurrencyID)
		if err != nil {
			return nil, err
		}

		// Check if mapping already exists
		existing, err := s.countryCurrencyRepo.FindByCountryAndCurrency(
			ctx,
			countryID,
			item.CurrencyID,
		)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			// Skip if already exists
			continue
		}

		// Create mapping
		mapping := &entity.CountryCurrency{
			CountryID:  countryID,
			CurrencyID: item.CurrencyID,
			IsPrimary:  item.IsPrimary,
		}

		if err := s.countryCurrencyRepo.Create(ctx, mapping); err != nil {
			return nil, err
		}

		response := factory.BuildCountryCurrencySimpleResponse(mapping)
		responses = append(responses, *response)
	}

	return responses, nil
}

// UpdateCountryCurrency updates a country-currency mapping (e.g., set as primary)
func (s *CountryCurrencyServiceImpl) UpdateCountryCurrency(
	ctx context.Context,
	countryID, currencyID uint,
	req model.CountryCurrencyUpdateRequest,
) (*model.CountryCurrencySimpleResponse, error) {
	// Verify country exists
	_, err := s.countryRepo.FindByID(ctx, countryID)
	if err != nil {
		return nil, err
	}

	// Verify mapping exists
	mapping, err := s.countryCurrencyRepo.FindByCountryAndCurrency(ctx, countryID, currencyID)
	if err != nil {
		return nil, err
	}
	if mapping == nil {
		return nil, userErrors.ErrCountryCurrencyNotFound
	}

	// If setting as primary, clear other primaries first
	if req.IsPrimary != nil && *req.IsPrimary {
		if err := s.countryCurrencyRepo.ClearPrimaryForCountry(ctx, countryID); err != nil {
			return nil, err
		}
	}

	// Update the mapping
	if req.IsPrimary != nil {
		mapping.IsPrimary = *req.IsPrimary
	}

	if err := s.countryCurrencyRepo.Update(ctx, mapping); err != nil {
		return nil, err
	}

	// Build response
	response := factory.BuildCountryCurrencySimpleResponse(mapping)
	return response, nil
}

// RemoveCurrencyFromCountry removes a currency from a country
func (s *CountryCurrencyServiceImpl) RemoveCurrencyFromCountry(
	ctx context.Context,
	countryID, currencyID uint,
) error {
	// Verify country exists
	_, err := s.countryRepo.FindByID(ctx, countryID)
	if err != nil {
		return err
	}

	// Verify mapping exists
	mapping, err := s.countryCurrencyRepo.FindByCountryAndCurrency(ctx, countryID, currencyID)
	if err != nil {
		return err
	}
	if mapping == nil {
		return userErrors.ErrCountryCurrencyNotFound
	}

	// Delete the mapping
	return s.countryCurrencyRepo.Delete(ctx, countryID, currencyID)
}
