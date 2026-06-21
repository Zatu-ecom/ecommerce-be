package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/error"

	"gorm.io/gorm"
)

// PackageOptionRepository defines the interface for package option data operations
type PackageOptionRepository interface {
	Create(ctx context.Context, packageOption *entity.PackageOption) error
	BulkCreate(ctx context.Context, packageOptions []entity.PackageOption) error
	Update(ctx context.Context, packageOption *entity.PackageOption) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*entity.PackageOption, error)
	FindAllByProductID(ctx context.Context, productID uint) ([]entity.PackageOption, error)
	DeleteByProductID(ctx context.Context, productID uint) error
}

// PackageOptionRepositoryImpl implements the PackageOptionRepository interface
type PackageOptionRepositoryImpl struct{}

// NewPackageOptionRepository creates a new instance of PackageOptionRepository
func NewPackageOptionRepository() PackageOptionRepository {
	return &PackageOptionRepositoryImpl{}
}

// Create creates a new package option
func (r *PackageOptionRepositoryImpl) Create(
	ctx context.Context,
	packageOption *entity.PackageOption,
) error {
	return db.DB(ctx).Create(packageOption).Error
}

// BulkCreate creates multiple package options in a single INSERT query
func (r *PackageOptionRepositoryImpl) BulkCreate(
	ctx context.Context,
	packageOptions []entity.PackageOption,
) error {
	if len(packageOptions) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&packageOptions).Error
}

// Update updates an existing package option
func (r *PackageOptionRepositoryImpl) Update(
	ctx context.Context,
	packageOption *entity.PackageOption,
) error {
	return db.DB(ctx).Save(packageOption).Error
}

// Delete deletes a package option by ID
func (r *PackageOptionRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := db.DB(ctx).Delete(&entity.PackageOption{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return prodErrors.ErrPackageOptionNotFound
	}
	return nil
}

// FindByID finds a package option by ID
func (r *PackageOptionRepositoryImpl) FindByID(
	ctx context.Context,
	id uint,
) (*entity.PackageOption, error) {
	var packageOption entity.PackageOption
	err := db.DB(ctx).First(&packageOption, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, prodErrors.ErrPackageOptionNotFound
		}
		return nil, err
	}
	return &packageOption, nil
}

// FindAllByProductID finds all package options for a given product
func (r *PackageOptionRepositoryImpl) FindAllByProductID(
	ctx context.Context,
	productID uint,
) ([]entity.PackageOption, error) {
	var packageOptions []entity.PackageOption
	err := db.DB(ctx).Where("product_id = ?", productID).Order("id ASC").Find(&packageOptions).Error
	if err != nil {
		return nil, err
	}
	return packageOptions, nil
}

// DeleteByProductID deletes all package options for a given product
func (r *PackageOptionRepositoryImpl) DeleteByProductID(
	ctx context.Context,
	productID uint,
) error {
	return db.DB(ctx).Where("product_id = ?", productID).Delete(&entity.PackageOption{}).Error
}
