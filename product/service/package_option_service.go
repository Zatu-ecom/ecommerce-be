package service

import (
	"context"

	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/error"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repository"
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
}

// NewPackageOptionService creates a new instance of PackageOptionService
func NewPackageOptionService(
	packageOptionRepo repository.PackageOptionRepository,
	productRepo repository.ProductRepository,
	validatorService ProductValidatorService,
) PackageOptionService {
	return &PackageOptionServiceImpl{
		packageOptionRepo: packageOptionRepo,
		productRepo:       productRepo,
		validatorService:  validatorService,
	}
}

// AddPackageOption adds a new package option to a product
func (s *PackageOptionServiceImpl) AddPackageOption(
	ctx context.Context,
	productID uint,
	sellerID uint,
	req model.PackageOptionCreateRequest,
) (*model.PackageOptionResponse, error) {
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	packageOption := factory.BuildPackageOptionFromCreateRequest(productID, req)
	if err := s.packageOptionRepo.Create(ctx, packageOption); err != nil {
		return nil, err
	}

	created, err := s.packageOptionRepo.FindByID(ctx, packageOption.ID)
	if err != nil {
		return nil, err
	}

	return factory.BuildPackageOptionResponse(created), nil
}

// UpdatePackageOption updates an existing package option
func (s *PackageOptionServiceImpl) UpdatePackageOption(
	ctx context.Context,
	productID uint,
	packageOptionID uint,
	sellerID uint,
	req model.PackageOptionUpdateRequest,
) (*model.PackageOptionResponse, error) {
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
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

	factory.ApplyPackageOptionUpdate(packageOption, req)
	if err := s.packageOptionRepo.Update(ctx, packageOption); err != nil {
		return nil, err
	}

	updated, err := s.packageOptionRepo.FindByID(ctx, packageOptionID)
	if err != nil {
		return nil, err
	}

	return factory.BuildPackageOptionResponse(updated), nil
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
	_, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, prodErrors.ErrProductNotFound
	}

	packageOptions, err := s.packageOptionRepo.FindAllByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}

	return factory.BuildPackageOptionsListResponse(packageOptions), nil
}

// BulkUpdatePackageOptions updates multiple package options for a product
func (s *PackageOptionServiceImpl) BulkUpdatePackageOptions(
	ctx context.Context,
	productID uint,
	sellerID uint,
	req model.BulkUpdatePackageOptionsRequest,
) (*model.BulkUpdatePackageOptionsResponse, error) {
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
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

		factory.ApplyBulkPackageOptionUpdate(packageOption, item)
		if err := s.packageOptionRepo.Update(ctx, packageOption); err != nil {
			return nil, err
		}

		updatedOptions = append(updatedOptions, *packageOption)
		updatedCount++
	}

	return &model.BulkUpdatePackageOptionsResponse{
		UpdatedCount:   updatedCount,
		PackageOptions: factory.BuildPackageOptionResponses(updatedOptions),
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

	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	packageOptions := factory.CreatePackageOptionsFromRequests(productID, requests)
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
