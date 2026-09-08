package service

import (
	"context"

	commonError "ecommerce-be/common/error"
	commonModel "ecommerce-be/common/model"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/error"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repository"
	userFactory "ecommerce-be/user/factory"
	userService "ecommerce-be/user/service"
)

// PackageOptionService defines the interface for package option business logic
type PackageOptionService interface {
	AddPackageOption(
		ctx context.Context,
		productID uint,
		sellerID uint,
		req model.PackageOptionCreateRequest,
	) (*model.PackageOptionResponse, error)

	UpdatePackageOption(
		ctx context.Context,
		productID uint,
		packageOptionID uint,
		sellerID uint,
		req model.PackageOptionUpdateRequest,
	) (*model.PackageOptionResponse, error)

	DeletePackageOption(
		ctx context.Context,
		productID uint,
		packageOptionID uint,
		sellerID uint,
	) error

	GetPackageOptions(
		ctx context.Context,
		productID uint,
	) (*model.PackageOptionsResponse, error)

	BulkUpdatePackageOptions(
		ctx context.Context,
		productID uint,
		sellerID uint,
		req model.BulkUpdatePackageOptionsRequest,
	) (*model.BulkUpdatePackageOptionsResponse, error)

	CreatePackageOptionsBulk(
		ctx context.Context,
		productID uint,
		sellerID uint,
		requests []model.PackageOptionRequest,
	) ([]entity.PackageOption, error)

	DeletePackageOptionsByProductID(ctx context.Context, productID uint) error
}

// PackageOptionServiceImpl implements the PackageOptionService interface
type PackageOptionServiceImpl struct {
	packageOptionRepo repository.PackageOptionRepository
	productRepo       repository.ProductRepository
	validatorService  ProductValidatorService
	userSvc           userService.UserService
}

// NewPackageOptionService creates a new instance of PackageOptionService
func NewPackageOptionService(
	packageOptionRepo repository.PackageOptionRepository,
	productRepo repository.ProductRepository,
	validatorService ProductValidatorService,
	userSvc userService.UserService,
) PackageOptionService {
	return &PackageOptionServiceImpl{
		packageOptionRepo: packageOptionRepo,
		productRepo:       productRepo,
		validatorService:  validatorService,
		userSvc:           userSvc,
	}
}

// sellerCurrency resolves the seller's base currency for price interpretation.
func (s *PackageOptionServiceImpl) sellerCurrency(ctx context.Context, sellerID uint) (commonModel.CurrencyInfo, error) {
	ccy, err := s.userSvc.GetSellerDefaultCurrency(ctx, sellerID)
	if err != nil {
		return commonModel.CurrencyInfo{}, err
	}
	return userFactory.ToCurrencyInfo(ccy), nil
}

// resolveSellerCurrency resolves the currency for a price write. When the token
// seller is 0 (admin acting on behalf of a seller), it falls back to the product
// owner's sellerID so the write is interpreted in the product's base currency.
func (s *PackageOptionServiceImpl) resolveSellerCurrency(
	ctx context.Context,
	product *entity.Product,
	tokenSellerID uint,
) (commonModel.CurrencyInfo, error) {
	sellerID := tokenSellerID
	if sellerID == 0 && product != nil {
		sellerID = product.SellerID
	}
	return s.sellerCurrency(ctx, sellerID)
}

// AddPackageOption adds a new package option to a product
func (s *PackageOptionServiceImpl) AddPackageOption(
	ctx context.Context,
	productID uint,
	sellerID uint,
	req model.PackageOptionCreateRequest,
) (*model.PackageOptionResponse, error) {
	product, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	ccy, err := s.resolveSellerCurrency(ctx, product, sellerID)
	if err != nil {
		return nil, err
	}

	packageOption, err := factory.BuildPackageOptionFromCreateRequest(productID, req, ccy)
	if err != nil {
		return nil, commonError.ErrValidation.WithMessage(err.Error())
	}
	if err := s.packageOptionRepo.Create(ctx, packageOption); err != nil {
		return nil, err
	}

	created, err := s.packageOptionRepo.FindByID(ctx, packageOption.ID)
	if err != nil {
		return nil, err
	}

	return factory.BuildPackageOptionResponse(created, ccy), nil
}

// UpdatePackageOption updates an existing package option
func (s *PackageOptionServiceImpl) UpdatePackageOption(
	ctx context.Context,
	productID uint,
	packageOptionID uint,
	sellerID uint,
	req model.PackageOptionUpdateRequest,
) (*model.PackageOptionResponse, error) {
	product, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	packageOption, err := s.packageOptionRepo.FindByID(ctx, packageOptionID)
	if err != nil {
		return nil, err
	}

	if packageOption.ProductID != productID {
		return nil, prodErrors.ErrPackageOptionNotFound
	}

	ccy, err := s.resolveSellerCurrency(ctx, product, sellerID)
	if err != nil {
		return nil, err
	}

	if err := factory.ApplyPackageOptionUpdate(packageOption, req, ccy); err != nil {
		return nil, commonError.ErrValidation.WithMessage(err.Error())
	}
	if err := s.packageOptionRepo.Update(ctx, packageOption); err != nil {
		return nil, err
	}

	updated, err := s.packageOptionRepo.FindByID(ctx, packageOptionID)
	if err != nil {
		return nil, err
	}

	return factory.BuildPackageOptionResponse(updated, ccy), nil
}

// DeletePackageOption removes a package option from a product
func (s *PackageOptionServiceImpl) DeletePackageOption(
	ctx context.Context,
	productID uint,
	packageOptionID uint,
	sellerID uint,
) error {
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return err
	}

	packageOption, err := s.packageOptionRepo.FindByID(ctx, packageOptionID)
	if err != nil {
		return err
	}

	if packageOption.ProductID != productID {
		return prodErrors.ErrPackageOptionNotFound
	}

	return s.packageOptionRepo.Delete(ctx, packageOptionID)
}

// GetPackageOptions retrieves all package options for a product
func (s *PackageOptionServiceImpl) GetPackageOptions(
	ctx context.Context,
	productID uint,
) (*model.PackageOptionsResponse, error) {
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, prodErrors.ErrProductNotFound
	}

	packageOptions, err := s.packageOptionRepo.FindAllByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}

	ccy, err := s.sellerCurrency(ctx, product.SellerID)
	if err != nil {
		return nil, err
	}

	return factory.BuildPackageOptionsListResponse(packageOptions, ccy), nil
}

// BulkUpdatePackageOptions updates multiple package options for a product
func (s *PackageOptionServiceImpl) BulkUpdatePackageOptions(
	ctx context.Context,
	productID uint,
	sellerID uint,
	req model.BulkUpdatePackageOptionsRequest,
) (*model.BulkUpdatePackageOptionsResponse, error) {
	product, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	ccy, err := s.resolveSellerCurrency(ctx, product, sellerID)
	if err != nil {
		return nil, err
	}

	updatedOptions := make([]entity.PackageOption, 0, len(req.PackageOptions))
	updatedCount := 0

	for _, item := range req.PackageOptions {
		packageOption, err := s.packageOptionRepo.FindByID(ctx, item.PackageOptionID)
		if err != nil {
			continue
		}

		if packageOption.ProductID != productID {
			continue
		}

		if err := factory.ApplyBulkPackageOptionUpdate(packageOption, item, ccy); err != nil {
			return nil, commonError.ErrValidation.WithMessage(err.Error())
		}
		if err := s.packageOptionRepo.Update(ctx, packageOption); err != nil {
			return nil, err
		}

		updatedOptions = append(updatedOptions, *packageOption)
		updatedCount++
	}

	return &model.BulkUpdatePackageOptionsResponse{
		UpdatedCount:   updatedCount,
		PackageOptions: factory.BuildPackageOptionResponses(updatedOptions, ccy),
	}, nil
}

// CreatePackageOptionsBulk creates multiple package options during product creation
func (s *PackageOptionServiceImpl) CreatePackageOptionsBulk(
	ctx context.Context,
	productID uint,
	sellerID uint,
	requests []model.PackageOptionRequest,
) ([]entity.PackageOption, error) {
	if len(requests) == 0 {
		return []entity.PackageOption{}, nil
	}

	product, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	ccy, err := s.resolveSellerCurrency(ctx, product, sellerID)
	if err != nil {
		return nil, err
	}

	packageOptions, err := factory.CreatePackageOptionsFromRequests(productID, requests, ccy)
	if err != nil {
		return nil, commonError.ErrValidation.WithMessage(err.Error())
	}
	if err := s.packageOptionRepo.BulkCreate(ctx, packageOptions); err != nil {
		return nil, err
	}

	return packageOptions, nil
}

// DeletePackageOptionsByProductID deletes all package options for a product
func (s *PackageOptionServiceImpl) DeletePackageOptionsByProductID(
	ctx context.Context,
	productID uint,
) error {
	return s.packageOptionRepo.DeleteByProductID(ctx, productID)
}
