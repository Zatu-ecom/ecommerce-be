package product

import (
	"ecommerce-be/product/entity"
)

type ProductAutoMigrate struct{}

func (p *ProductAutoMigrate) AutoMigrate() []interface{} {
	return []interface{}{
		&entity.Category{},
		&entity.AttributeDefinition{},
		&entity.CategoryAttribute{},
		&entity.Product{},
		&entity.ProductAttribute{},
		&entity.ProductOption{},
		&entity.ProductOptionValue{},
		&entity.ProductVariant{},
		&entity.VariantOptionValue{},
		&entity.PackageOption{},
	}
}

func NewProductAutoMigrate() *ProductAutoMigrate {
	return &ProductAutoMigrate{}
}
