package service

import (
	"context"
	"time"

	"ecommerce-be/common/constants"
	commonEntity "ecommerce-be/common/db"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/filegateway"
	db "ecommerce-be/common/db"
	fileGateway "ecommerce-be/file/gateway"
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

	GetProfile(
		ctx context.Context,
		userID uint,
	) (*model.SellerFullProfileResponse, error)
}

// SellerServiceImpl implements the SellerService interface
type SellerServiceImpl struct {
	userService           UserService
	sellerSettingsService SellerSettingsService
	sellerProfileRepo     repository.SellerProfileRepository
	userRepo              repository.UserRepository
	fileGateway           filegateway.FileDisplayGateway
}

// NewSellerService creates a new instance of SellerService
func NewSellerService(
	userService UserService,
	sellerSettingsService SellerSettingsService,
	userRepo repository.UserRepository,
	sellerProfileRepo repository.SellerProfileRepository,
	fileGateway filegateway.FileDisplayGateway,
) SellerService {
	return &SellerServiceImpl{
		userService:           userService,
		sellerSettingsService: sellerSettingsService,
		userRepo:              userRepo,
		sellerProfileRepo:     sellerProfileRepo,
		fileGateway:           fileGateway,
	}
}

func (s *SellerServiceImpl) RegisterSeller(
	ctx context.Context,
	req model.SellerRegisterRequest,
) (*model.SellerRegisterResponse, error) {
	if err := s.validateSellerData(ctx, req); err != nil {
		return nil, err
	}

	return db.WithTransactionResult(
		ctx,
		func(txCtx context.Context) (*model.SellerRegisterResponse, error) {
			return s.executeRegistration(txCtx, req)
		},
	)
}

func (s *SellerServiceImpl) validateSellerData(
	ctx context.Context,
	req model.SellerRegisterRequest,
) error {
	if req.User.Password != req.User.ConfirmPassword {
		return userErrors.ErrPasswordMismatch
	}

	if req.Profile.TaxID != "" {
		exists, err := s.sellerProfileRepo.ExistsByTaxID(ctx, req.Profile.TaxID)
		if err != nil {
			return userErrors.ErrTaxIDCheckFailed
		}
		if exists {
			return userErrors.ErrTaxIDAlreadyExists
		}
	}

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

func (s *SellerServiceImpl) executeRegistration(
	ctx context.Context,
	req model.SellerRegisterRequest,
) (*model.SellerRegisterResponse, error) {
	user, role, err := s.userService.CreateUserWithRole(
		ctx,
		req.User.CreateUserRequest,
		constants.SELLER_ROLE_NAME,
	)
	if err != nil {
		return nil, userErrors.ErrUserCreateFailed
	}

	logoFileID := req.Profile.BusinessLogoFileID
	if _, err := filegateway.ResolveSingle(ctx, s.fileGateway, logoFileID, &user.ID); err != nil {
		if fileGateway.IsFileNotFound(err) || err == commonError.ErrFileNotAccessible {
			return nil, userErrors.ErrInvalidBusinessLogoFile
		}
		return nil, err
	}

	now := time.Now()
	profile := &entity.SellerProfile{
		UserID:             user.ID,
		BusinessName:       req.Profile.BusinessName,
		BusinessLogoFileID: &logoFileID,
		TaxID:              req.Profile.TaxID,
		IsVerified:         false,
		BaseEntityWithoutID: commonEntity.BaseEntityWithoutID{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	if err := s.sellerProfileRepo.Create(ctx, profile); err != nil {
		return nil, userErrors.ErrProfileCreateFailed
	}

	user.SellerID = user.ID
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, userErrors.ErrSellerIDUpdateFailed
	}

	var settings *model.SellerSettingsResponse
	requiresOnboarding := true

	if req.Settings != nil {
		settings, err = s.sellerSettingsService.Create(ctx, user.ID, req.Settings)
		if err != nil {
			return nil, err
		}
		requiresOnboarding = false
	}

	authResponse, err := factory.BuildAuthResponse(user, role, &user.ID, nil)
	if err != nil {
		return nil, userErrors.ErrTokenGenerationFailed
	}

	logo := filegateway.ResolveOptional(ctx, s.fileGateway, profile.BusinessLogoFileID, &user.ID)
	return factory.BuildSellerRegisterResponse(
		user,
		profile,
		settings,
		authResponse.Token,
		authResponse.ExpiresIn,
		requiresOnboarding,
		logo,
	), nil
}

func (s *SellerServiceImpl) GetProfile(
	ctx context.Context,
	userID uint,
) (*model.SellerFullProfileResponse, error) {
	user, err := s.userService.GetProfile(ctx, userID)
	if err != nil {
		return nil, userErrors.ErrUserNotFound
	}

	profile, err := s.sellerProfileRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, userErrors.ErrSellerProfileNotFound
	}

	settings, _ := s.sellerSettingsService.GetBySellerID(ctx, userID)

	logo := filegateway.ResolveOptional(ctx, s.fileGateway, profile.BusinessLogoFileID, &userID)
	return factory.BuildSellerFullProfileResponse(
		user.UserResponse,
		profile,
		settings,
		user.Addresses,
		logo,
	), nil
}
