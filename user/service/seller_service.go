package service

import (
	"context"
	"time"

	"ecommerce-be/common/constants"
	commonEntity "ecommerce-be/common/db"
	db "ecommerce-be/common/db"
	"ecommerce-be/user/entity"
	userErrors "ecommerce-be/user/error"
	"ecommerce-be/user/factory"
	"ecommerce-be/user/model"
	"ecommerce-be/user/repository"
)

// SellerService defines the interface for seller registration operations
type SellerService interface {
	RegisterSeller(
		ctx context.Context,
		req model.SellerRegisterRequest,
	) (*model.SellerRegisterResponse, error)

	// GetProfile retrieves the full seller profile (user + profile + settings)
	GetProfile(
		ctx context.Context,
		userID uint,
	) (*model.SellerFullProfileResponse, error)
}

// SellerServiceImpl implements the SellerService interface
type SellerServiceImpl struct {
	// Services (following SOLID - use services to reuse business logic)
	userService           UserService
	sellerSettingsService SellerSettingsService

	// Repositories (for operations managed by this service)
	sellerProfileRepo repository.SellerProfileRepository
	userRepo          repository.UserRepository
}

// NewSellerService creates a new instance of SellerService
func NewSellerService(
	userService UserService,
	sellerSettingsService SellerSettingsService,
	userRepo repository.UserRepository,
	sellerProfileRepo repository.SellerProfileRepository,
) SellerService {
	return &SellerServiceImpl{
		userService:           userService,
		sellerSettingsService: sellerSettingsService,
		userRepo:              userRepo,
		sellerProfileRepo:     sellerProfileRepo,
	}
}

// RegisterSeller creates a new seller account with user, profile, and optional settings
// Uses transaction to ensure atomicity across multiple table operations
func (s *SellerServiceImpl) RegisterSeller(
	ctx context.Context,
	req model.SellerRegisterRequest,
) (*model.SellerRegisterResponse, error) {
	// 1. Validate seller-specific data BEFORE starting transaction
	if err := s.validateSellerData(ctx, req); err != nil {
		return nil, err
	}

	// 2. Execute registration within a transaction
	return db.WithTransactionResult(
		ctx,
		func(txCtx context.Context) (*model.SellerRegisterResponse, error) {
			return s.executeRegistration(txCtx, req)
		},
	)
}

// validateSellerData validates seller-specific data before transaction
func (s *SellerServiceImpl) validateSellerData(
	ctx context.Context,
	req model.SellerRegisterRequest,
) error {
	// Validate password confirmation
	if req.User.Password != req.User.ConfirmPassword {
		return userErrors.ErrPasswordMismatch
	}

	// Validate TaxID uniqueness (if provided)
	if req.Profile.TaxID != "" {
		exists, err := s.sellerProfileRepo.ExistsByTaxID(ctx, req.Profile.TaxID)
		if err != nil {
			return userErrors.ErrTaxIDCheckFailed
		}
		if exists {
			return userErrors.ErrTaxIDAlreadyExists
		}
	}

	// Validate settings if provided (using SellerSettingsService)
	if req.Settings != nil {
		if err := s.sellerSettingsService.ValidateSettingsData(
			ctx,
			req.Settings.BusinessCountryID,
			req.Settings.BaseCurrencyID,
			req.Settings.SettlementCurrencyID,
		); err != nil {
			return err
		}
	}

	return nil
}

// executeRegistration performs the actual registration within a transaction
func (s *SellerServiceImpl) executeRegistration(
	ctx context.Context,
	req model.SellerRegisterRequest,
) (*model.SellerRegisterResponse, error) {
	// 1. Create user using UserService (reuses email validation, password hashing, etc.)

	user, role, err := s.userService.CreateUserWithRole(
		ctx,
		req.User.CreateUserRequest,
		constants.SELLER_ROLE_NAME,
	)
	if err != nil {
		return nil, userErrors.ErrUserCreateFailed
	}

	// 2. Update user's SellerID to point to itself (seller is their own seller)
	user.SellerID = user.ID
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, userErrors.ErrSellerIDUpdateFailed
	}

	// 3. Create seller profile (this service's responsibility)
	now := time.Now()
	profile := &entity.SellerProfile{
		UserID:       user.ID,
		BusinessName: req.Profile.BusinessName,
		BusinessLogo: req.Profile.BusinessLogo,
		TaxID:        req.Profile.TaxID,
		IsVerified:   false,
		BaseEntityWithoutID: commonEntity.BaseEntityWithoutID{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	if err := s.sellerProfileRepo.Create(ctx, profile); err != nil {
		return nil, userErrors.ErrProfileCreateFailed
	}

	// 4. Create seller settings if provided (using SellerSettingsService)
	// Settings is already SellerSettingsCreateRequest - pass directly
	var settings *model.SellerSettingsResponse
	requiresOnboarding := true

	if req.Settings != nil {
		settings, err = s.sellerSettingsService.Create(ctx, user.ID, req.Settings)
		if err != nil {
			return nil, err // Error already wrapped by SellerSettingsService
		}
		requiresOnboarding = false
	}

	// 5. Generate JWT token
	authResponse, err := factory.BuildAuthResponse(user, role, &user.ID)
	if err != nil {
		return nil, userErrors.ErrTokenGenerationFailed
	}

	// 6. Build and return response using factory builders
	return factory.BuildSellerRegisterResponse(
		user,
		profile,
		settings,
		authResponse.Token,
		authResponse.ExpiresIn,
		requiresOnboarding,
	), nil
}

// GetProfile retrieves the full seller profile including user, profile, and settings
func (s *SellerServiceImpl) GetProfile(
	ctx context.Context,
	userID uint,
) (*model.SellerFullProfileResponse, error) {
	// 1. Get user (using UserService - following SOLID)
	user, err := s.userService.GetProfile(ctx, userID)
	if err != nil {
		return nil, userErrors.ErrUserNotFound
	}

	// 2. Get seller profile
	profile, err := s.sellerProfileRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, userErrors.ErrSellerProfileNotFound
	}

	// 3. Get seller settings (may not exist if onboarding incomplete)
	// SellerID is the same as UserID for sellers
	settings, _ := s.sellerSettingsService.GetBySellerID(ctx, userID)

	// 4. Build and return response
	return factory.BuildSellerFullProfileResponse(
		user.UserResponse,
		profile,
		settings,
		user.Addresses,
	), nil
}
