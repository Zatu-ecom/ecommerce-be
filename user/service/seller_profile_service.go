package service

import (
	"context"

	"ecommerce-be/common/filegateway"
	commonError "ecommerce-be/common/error"
	fileGateway "ecommerce-be/file/gateway"
	userErrors "ecommerce-be/user/error"
	"ecommerce-be/user/factory"
	"ecommerce-be/user/model"
	"ecommerce-be/user/repository"
)

// SellerProfileService defines the interface for seller profile operations
type SellerProfileService interface {
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
	fileGateway           filegateway.FileDisplayGateway
}

// NewSellerProfileService creates a new instance of SellerProfileService
func NewSellerProfileService(
	userRepo repository.UserRepository,
	sellerProfileRepo repository.SellerProfileRepository,
	sellerSettingsService SellerSettingsService,
	fileGateway filegateway.FileDisplayGateway,
) SellerProfileService {
	return &SellerProfileServiceImpl{
		userRepo:              userRepo,
		sellerProfileRepo:     sellerProfileRepo,
		sellerSettingsService: sellerSettingsService,
		fileGateway:           fileGateway,
	}
}

func (s *SellerProfileServiceImpl) UpdateProfile(
	ctx context.Context,
	userID uint,
	req model.SellerProfileUpdateRequest,
) (*model.SellerProfileResponse, error) {
	profile, err := s.sellerProfileRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, userErrors.ErrSellerProfileNotFound
	}

	if req.TaxID != nil && *req.TaxID != profile.TaxID {
		exists, err := s.sellerProfileRepo.ExistsByTaxIDExcluding(ctx, *req.TaxID, userID)
		if err != nil {
			return nil, userErrors.ErrTaxIDCheckFailed
		}
		if exists {
			return nil, userErrors.ErrTaxIDAlreadyExists
		}
	}

	if req.BusinessName != nil {
		profile.BusinessName = *req.BusinessName
	}
	if req.BusinessLogoFileID != nil {
		if *req.BusinessLogoFileID != "" {
			if _, err := filegateway.ResolveSingle(ctx, s.fileGateway, *req.BusinessLogoFileID, &userID); err != nil {
				if fileGateway.IsFileNotFound(err) || err == commonError.ErrFileNotAccessible {
					return nil, userErrors.ErrInvalidBusinessLogoFile
				}
				return nil, err
			}
		}
		profile.BusinessLogoFileID = req.BusinessLogoFileID
	}
	if req.TaxID != nil {
		profile.TaxID = *req.TaxID
	}

	if err := s.sellerProfileRepo.Update(ctx, profile); err != nil {
		return nil, userErrors.ErrProfileUpdateFailed
	}

	logo := filegateway.ResolveOptional(ctx, s.fileGateway, profile.BusinessLogoFileID, &userID)
	return factory.BuildSellerProfileResponsePtr(profile, logo), nil
}
