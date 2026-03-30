package service

import (
	"context"

	userErrors "ecommerce-be/user/error"
	"ecommerce-be/user/factory"
	"ecommerce-be/user/model"
	"ecommerce-be/user/repository"
)

// SellerProfileService defines the interface for seller profile operations
type SellerProfileService interface {
	// UpdateProfile updates seller profile
	UpdateProfile(
		ctx context.Context,
		userID uint,
		req model.SellerProfileUpdateRequest,
	) (*model.SellerProfileResponse, error)
}

// SellerProfileServiceImpl implements the SellerProfileService interface
type SellerProfileServiceImpl struct {
	userRepo              repository.UserRepository
	sellerProfileRepo     repository.SellerProfileRepository
	sellerSettingsService SellerSettingsService
}

// NewSellerProfileService creates a new instance of SellerProfileService
func NewSellerProfileService(
	userRepo repository.UserRepository,
	sellerProfileRepo repository.SellerProfileRepository,
	sellerSettingsService SellerSettingsService,
) SellerProfileService {
	return &SellerProfileServiceImpl{
		userRepo:              userRepo,
		sellerProfileRepo:     sellerProfileRepo,
		sellerSettingsService: sellerSettingsService,
	}
}

// UpdateProfile updates the seller profile
func (s *SellerProfileServiceImpl) UpdateProfile(
	ctx context.Context,
	userID uint,
	req model.SellerProfileUpdateRequest,
) (*model.SellerProfileResponse, error) {
	// 1. Get existing profile
	profile, err := s.sellerProfileRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, userErrors.ErrSellerProfileNotFound
	}

	// 2. Validate TaxID uniqueness if changed
	if req.TaxID != nil && *req.TaxID != profile.TaxID {
		exists, err := s.sellerProfileRepo.ExistsByTaxIDExcluding(ctx, *req.TaxID, userID)
		if err != nil {
			return nil, userErrors.ErrTaxIDCheckFailed
		}
		if exists {
			return nil, userErrors.ErrTaxIDAlreadyExists
		}
	}

	// 3. Update fields if provided
	if req.BusinessName != nil {
		profile.BusinessName = *req.BusinessName
	}
	if req.BusinessLogo != nil {
		profile.BusinessLogo = *req.BusinessLogo
	}
	if req.TaxID != nil {
		profile.TaxID = *req.TaxID
	}

	// 4. Save to database
	if err := s.sellerProfileRepo.Update(ctx, profile); err != nil {
		return nil, userErrors.ErrProfileUpdateFailed
	}

	// 5. Build and return response
	return factory.BuildSellerProfileResponsePtr(profile), nil
}
