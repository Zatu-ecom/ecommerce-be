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

// CurrencyService defines the interface for currency-related business logic
type CurrencyService interface {
	// Query methods - used by both public and admin APIs
	GetAllCurrencies(
		ctx context.Context,
		filter model.CurrencyQueryParams,
		includeInactive bool,
	) (*model.CurrencyListResponse, error)

	GetCurrencyByID(
		ctx context.Context,
		id uint,
	) (*model.CurrencyDetailResponse, error)

	// Mutation methods - admin only
	CreateCurrency(
		ctx context.Context,
		req model.CurrencyCreateRequest,
	) (*model.CurrencyResponse, error)

	UpdateCurrency(
		ctx context.Context,
		id uint,
		req model.CurrencyUpdateRequest,
	) (*model.CurrencyResponse, error)

	DeleteCurrency(ctx context.Context, id uint) error
}

// CurrencyServiceImpl implements the CurrencyService interface
type CurrencyServiceImpl struct {
	currencyRepo repository.CurrencyRepository
}

// NewCurrencyService creates a new instance of CurrencyService
func NewCurrencyService(
	currencyRepo repository.CurrencyRepository,
) *CurrencyServiceImpl {
	return &CurrencyServiceImpl{
		currencyRepo: currencyRepo,
	}
}

// GetAllCurrencies retrieves all currencies with optional filters
// includeInactive: false for public API (only active), true for admin API
func (s *CurrencyServiceImpl) GetAllCurrencies(
	ctx context.Context,
	filter model.CurrencyQueryParams,
	includeInactive bool,
) (*model.CurrencyListResponse, error) {
	// Set default pagination
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	// Get currencies from repository
	currencies, err := s.currencyRepo.FindAll(ctx, filter, includeInactive)
	if err != nil {
		return nil, err
	}

	// Get total count for pagination
	totalCount, err := s.currencyRepo.CountAll(ctx, filter, includeInactive)
	if err != nil {
		return nil, err
	}

	// Build response
	currencyResponses := factory.BuildCurrencyListResponse(currencies)

	// Calculate pagination values
	totalItems := int(totalCount)
	totalPages := helper.CalculateTotalPages(totalItems, filter.Limit)

	return &model.CurrencyListResponse{
		Currencies: currencyResponses,
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

// GetCurrencyByID retrieves a currency by ID with its countries
func (s *CurrencyServiceImpl) GetCurrencyByID(
	ctx context.Context,
	id uint,
) (*model.CurrencyDetailResponse, error) {
	// Get currency with countries
	currency, err := s.currencyRepo.FindByIDWithCountries(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build detailed response
	response := factory.BuildCurrencyDetailResponse(currency)
	return &response, nil
}

// CreateCurrency creates a new currency (admin only)
func (s *CurrencyServiceImpl) CreateCurrency(
	ctx context.Context,
	req model.CurrencyCreateRequest,
) (*model.CurrencyResponse, error) {
	// Normalize code to uppercase
	code := strings.ToUpper(req.Code)

	// Check if currency with this code already exists
	existing, err := s.currencyRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, userErrors.ErrDuplicateCurrencyCode
	}

	// Create currency entity
	currency := &entity.Currency{
		Code:          code,
		Name:          req.Name,
		Symbol:        req.Symbol,
		SymbolNative:  req.SymbolNative,
		DecimalDigits: req.DecimalDigits,
		IsActive:      req.IsActive,
	}

	// Save to database
	if err := s.currencyRepo.Create(ctx, currency); err != nil {
		return nil, err
	}

	// Build response
	response := factory.BuildCurrencyResponse(currency)
	return &response, nil
}

// UpdateCurrency updates an existing currency (admin only)
func (s *CurrencyServiceImpl) UpdateCurrency(
	ctx context.Context,
	id uint,
	req model.CurrencyUpdateRequest,
) (*model.CurrencyResponse, error) {
	// Fetch existing currency
	currency, err := s.currencyRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// If code is being updated, check for duplicates
	if req.Code != nil {
		newCode := strings.ToUpper(*req.Code)
		if newCode != currency.Code {
			existing, err := s.currencyRepo.FindByCode(ctx, newCode)
			if err != nil {
				return nil, err
			}
			if existing != nil {
				return nil, userErrors.ErrDuplicateCurrencyCode
			}
			currency.Code = newCode
		}
	}

	// Update fields if provided
	if req.Name != nil {
		currency.Name = *req.Name
	}
	if req.Symbol != nil {
		currency.Symbol = *req.Symbol
	}
	if req.SymbolNative != nil {
		currency.SymbolNative = *req.SymbolNative
	}
	if req.DecimalDigits != nil {
		currency.DecimalDigits = *req.DecimalDigits
	}
	if req.IsActive != nil {
		currency.IsActive = *req.IsActive
	}

	// Save changes
	if err := s.currencyRepo.Update(ctx, currency); err != nil {
		return nil, err
	}

	// Build response
	response := factory.BuildCurrencyResponse(currency)
	return &response, nil
}

// DeleteCurrency deletes a currency (admin only)
func (s *CurrencyServiceImpl) DeleteCurrency(ctx context.Context, id uint) error {
	// Verify currency exists
	_, err := s.currencyRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete currency (soft delete via GORM)
	return s.currencyRepo.Delete(ctx, id)
}
