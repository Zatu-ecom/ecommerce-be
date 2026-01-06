package service

import (
	"context"
	"time"

	commonEntity "ecommerce-be/common/db"
	"ecommerce-be/user/entity"
	userErrors "ecommerce-be/user/error"
	"ecommerce-be/user/factory"
	"ecommerce-be/user/model"
	"ecommerce-be/user/repository"
)

// SellerSettingsService defines the interface for seller settings business logic
type SellerSettingsService interface {
	// Create creates new seller settings
	Create(
		ctx context.Context,
		sellerID uint,
		req *model.SellerSettingsCreateRequest,
	) (*model.SellerSettingsResponse, error)

	// GetBySellerID retrieves seller settings by seller ID
	GetBySellerID(
		ctx context.Context,
		sellerID uint,
	) (*model.SellerSettingsResponse, error)

	// Update updates existing seller settings
	Update(
		ctx context.Context,
		sellerID uint,
		req model.SellerSettingsUpdateRequest,
	) (*model.SellerSettingsResponse, error)

	// ValidateSettingsData validates country and currency IDs
	// Used before creating settings to ensure data integrity
	ValidateSettingsData(
		ctx context.Context,
		businessCountryID uint,
		baseCurrencyID uint,
		settlementCurrencyID *uint,
	) error

	// ExistsBySellerID checks if settings exist for a seller
	ExistsBySellerID(ctx context.Context, sellerID uint) (bool, error)
}

// SellerSettingsServiceImpl implements the SellerSettingsService interface
type SellerSettingsServiceImpl struct {
	settingsRepo    repository.SellerSettingsRepository
	countryService  CountryService
	currencyService CurrencyService
}

// NewSellerSettingsService creates a new instance of SellerSettingsService
func NewSellerSettingsService(
	settingsRepo repository.SellerSettingsRepository,
	countryService CountryService,
	currencyService CurrencyService,
) SellerSettingsService {
	return &SellerSettingsServiceImpl{
		settingsRepo:    settingsRepo,
		countryService:  countryService,
		currencyService: currencyService,
	}
}

// Create creates new seller settings
func (s *SellerSettingsServiceImpl) Create(
	ctx context.Context,
	sellerID uint,
	req *model.SellerSettingsCreateRequest,
) (*model.SellerSettingsResponse, error) {
	// Check if settings already exist
	exists, err := s.settingsRepo.ExistsBySellerID(ctx, sellerID)
	if err != nil {
		return nil, userErrors.ErrSellerSettingsExists
	}
	if exists {
		return nil, userErrors.ErrSellerSettingsExists
	}

	// Validate country and currency
	if err := s.ValidateSettingsData(ctx, req.BusinessCountryID, req.BaseCurrencyID, req.SettlementCurrencyID); err != nil {
		return nil, err
	}

	// Build entity
	now := time.Now()
	settings := &entity.SellerSettings{
		SellerID:                     sellerID,
		BusinessCountryID:            req.BusinessCountryID,
		BaseCurrencyID:               req.BaseCurrencyID,
		DisplayPricesInBuyerCurrency: false,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	// Set settlement currency (defaults to base currency if not provided)
	if req.SettlementCurrencyID != nil {
		settings.SettlementCurrencyID = *req.SettlementCurrencyID
	} else {
		settings.SettlementCurrencyID = req.BaseCurrencyID
	}

	// Set display preference if provided
	if req.DisplayPricesInBuyerCurrency != nil {
		settings.DisplayPricesInBuyerCurrency = *req.DisplayPricesInBuyerCurrency
	}

	// Save to database
	if err := s.settingsRepo.Create(ctx, settings); err != nil {
		return nil, userErrors.ErrSettingsCreateFailed
	}

	return factory.BuildSellerSettingsResponse(settings), nil
}

// GetBySellerID retrieves seller settings by seller ID
func (s *SellerSettingsServiceImpl) GetBySellerID(
	ctx context.Context,
	sellerID uint,
) (*model.SellerSettingsResponse, error) {
	settings, err := s.settingsRepo.FindBySellerID(ctx, sellerID)
	if err != nil {
		return nil, userErrors.ErrSellerSettingsNotFound
	}

	return factory.BuildSellerSettingsResponse(settings), nil
}

// Update updates existing seller settings
func (s *SellerSettingsServiceImpl) Update(
	ctx context.Context,
	sellerID uint,
	req model.SellerSettingsUpdateRequest,
) (*model.SellerSettingsResponse, error) {
	// Get existing settings
	settings, err := s.settingsRepo.FindBySellerID(ctx, sellerID)
	if err != nil {
		return nil, userErrors.ErrSellerSettingsNotFound
	}

	// Update fields if provided
	if req.BusinessCountryID != nil {
		// Validate country exists
		if _, err := s.countryService.GetCountryByID(ctx, *req.BusinessCountryID); err != nil {
			return nil, userErrors.ErrCountryNotFound
		}
		settings.BusinessCountryID = *req.BusinessCountryID
	}

	if req.BaseCurrencyID != nil {
		// Validate currency exists
		if _, err := s.currencyService.GetCurrencyByID(ctx, *req.BaseCurrencyID); err != nil {
			return nil, userErrors.ErrCurrencyNotFound
		}
		settings.BaseCurrencyID = *req.BaseCurrencyID
	}

	if req.SettlementCurrencyID != nil {
		// Validate settlement currency exists
		if _, err := s.currencyService.GetCurrencyByID(ctx, *req.SettlementCurrencyID); err != nil {
			return nil, userErrors.ErrCurrencyNotFound
		}
		settings.SettlementCurrencyID = *req.SettlementCurrencyID
	}

	if req.DisplayPricesInBuyerCurrency != nil {
		settings.DisplayPricesInBuyerCurrency = *req.DisplayPricesInBuyerCurrency
	}

	settings.UpdatedAt = time.Now()

	// Save changes
	if err := s.settingsRepo.Update(ctx, settings); err != nil {
		return nil, userErrors.ErrSellerSettingsExists // Generic update error
	}

	return factory.BuildSellerSettingsResponse(settings), nil
}

// ValidateSettingsData validates country and currency IDs using their respective services
func (s *SellerSettingsServiceImpl) ValidateSettingsData(
	ctx context.Context,
	businessCountryID uint,
	baseCurrencyID uint,
	settlementCurrencyID *uint,
) error {
	// Validate country exists using CountryService
	if _, err := s.countryService.GetCountryByID(ctx, businessCountryID); err != nil {
		return userErrors.ErrCountryNotFound
	}

	// Validate base currency exists using CurrencyService
	if _, err := s.currencyService.GetCurrencyByID(ctx, baseCurrencyID); err != nil {
		return userErrors.ErrCurrencyNotFound
	}

	// Validate settlement currency if provided
	if settlementCurrencyID != nil {
		if _, err := s.currencyService.GetCurrencyByID(ctx, *settlementCurrencyID); err != nil {
			return userErrors.ErrCurrencyNotFound
		}
	}

	return nil
}

// ExistsBySellerID checks if settings exist for a seller
func (s *SellerSettingsServiceImpl) ExistsBySellerID(
	ctx context.Context,
	sellerID uint,
) (bool, error) {
	return s.settingsRepo.ExistsBySellerID(ctx, sellerID)
}
