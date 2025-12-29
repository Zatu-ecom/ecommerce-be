package service

import (
	"context"
	"strings"

	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/user/entity"
	userErrors "ecommerce-be/user/error"
	"ecommerce-be/user/factory"
	"ecommerce-be/user/model"
	"ecommerce-be/user/repository"
)

// CountryService defines the interface for country-related business logic
type CountryService interface {
	// Query methods - used by both public and admin APIs
	GetAllCountries(
		ctx context.Context,
		filter model.CountryQueryParams,
		includeInactive bool,
	) (*model.CountryListResponse, error)

	GetCountryByID(
		ctx context.Context,
		id uint,
	) (*model.CountryDetailResponse, error)

	// Mutation methods - admin only
	CreateCountry(
		ctx context.Context,
		req model.CountryCreateRequest,
	) (*model.CountryResponse, error)

	UpdateCountry(
		ctx context.Context,
		id uint, req model.CountryUpdateRequest,
	) (*model.CountryResponse, error)

	DeleteCountry(ctx context.Context, id uint) error
}

// CountryServiceImpl implements the CountryService interface
type CountryServiceImpl struct {
	countryRepo repository.CountryRepository
}

// NewCountryService creates a new instance of CountryService
func NewCountryService(
	countryRepo repository.CountryRepository,
) *CountryServiceImpl {
	return &CountryServiceImpl{
		countryRepo: countryRepo,
	}
}

// GetAllCountries retrieves all countries with optional filters
// includeInactive: false for public API (only active), true for admin API
func (s *CountryServiceImpl) GetAllCountries(
	ctx context.Context,
	filter model.CountryQueryParams,
	includeInactive bool,
) (*model.CountryListResponse, error) {
	// Set default pagination
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	// Get countries from repository
	countries, err := s.countryRepo.FindAll(ctx, filter, includeInactive)
	if err != nil {
		return nil, err
	}

	// Get total count for pagination
	totalCount, err := s.countryRepo.CountAll(ctx, filter, includeInactive)
	if err != nil {
		return nil, err
	}

	// Build response
	countryResponses := factory.BuildCountryListResponse(countries)

	// Calculate pagination values
	totalItems := int(totalCount)
	totalPages := helper.CalculateTotalPages(totalItems, filter.Limit)

	return &model.CountryListResponse{
		Countries: countryResponses,
		Pagination: common.PaginationResponse{
			CurrentPage:  filter.Page,
			ItemsPerPage: filter.Limit,
			TotalItems:   totalItems,
			TotalPages:   totalPages,
			HasNext:      filter.Page < totalPages,
			HasPrev:      filter.Page > 1,
		},
	}, nil
}

// GetCountryByID retrieves a country by ID with its currencies
func (s *CountryServiceImpl) GetCountryByID(
	ctx context.Context,
	id uint,
) (*model.CountryDetailResponse, error) {
	// Get country with currencies
	country, err := s.countryRepo.FindByIDWithCurrencies(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build detailed response
	response := factory.BuildCountryDetailResponse(country)
	return &response, nil
}

// CreateCountry creates a new country (admin only)
func (s *CountryServiceImpl) CreateCountry(
	ctx context.Context,
	req model.CountryCreateRequest,
) (*model.CountryResponse, error) {
	// Normalize code to uppercase
	code := strings.ToUpper(req.Code)

	// Check if country with this code already exists
	existing, err := s.countryRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, userErrors.ErrDuplicateCountryCode
	}

	// Create country entity
	country := &entity.Country{
		Code:       code,
		CodeAlpha3: strings.ToUpper(req.CodeAlpha3),
		Name:       req.Name,
		NativeName: req.NativeName,
		PhoneCode:  req.PhoneCode,
		Region:     req.Region,
		FlagEmoji:  req.FlagEmoji,
		IsActive:   req.IsActive,
	}

	// Save to database
	if err := s.countryRepo.Create(ctx, country); err != nil {
		return nil, err
	}

	// Build response
	response := factory.BuildCountryResponse(country)
	return &response, nil
}

// UpdateCountry updates an existing country (admin only)
func (s *CountryServiceImpl) UpdateCountry(
	ctx context.Context,
	id uint,
	req model.CountryUpdateRequest,
) (*model.CountryResponse, error) {
	// Fetch existing country
	country, err := s.countryRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// If code is being updated, check for duplicates
	if req.Code != nil {
		newCode := strings.ToUpper(*req.Code)
		if newCode != country.Code {
			existing, err := s.countryRepo.FindByCode(ctx, newCode)
			if err != nil {
				return nil, err
			}
			if existing != nil {
				return nil, userErrors.ErrDuplicateCountryCode
			}
			country.Code = newCode
		}
	}

	// Update fields if provided
	if req.CodeAlpha3 != nil {
		country.CodeAlpha3 = strings.ToUpper(*req.CodeAlpha3)
	}
	if req.Name != nil {
		country.Name = *req.Name
	}
	if req.NativeName != nil {
		country.NativeName = *req.NativeName
	}
	if req.PhoneCode != nil {
		country.PhoneCode = *req.PhoneCode
	}
	if req.Region != nil {
		country.Region = *req.Region
	}
	if req.FlagEmoji != nil {
		country.FlagEmoji = *req.FlagEmoji
	}
	if req.IsActive != nil {
		country.IsActive = *req.IsActive
	}

	// Save changes
	if err := s.countryRepo.Update(ctx, country); err != nil {
		return nil, err
	}

	// Build response
	response := factory.BuildCountryResponse(country)
	return &response, nil
}

// DeleteCountry deletes a country (admin only)
func (s *CountryServiceImpl) DeleteCountry(ctx context.Context, id uint) error {
	// Verify country exists
	_, err := s.countryRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete country (soft delete via GORM)
	return s.countryRepo.Delete(ctx, id)
}
