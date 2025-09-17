package product

import (
	"ecommerce-be/product/entity"
)

type ProductAutoMigrate struct{}

func (p *ProductAutoMigrate) AutoMigrate() []interface{} {
	return []interface{}{
		&entity.Category{},
		&entity.Product{},
		&entity.AttributeDefinition{},
		&entity.CategoryAttribute{},
		&entity.ProductAttribute{},
		&entity.PackageOption{},
		&entity.Product{},
	}
}

func NewProductAutoMigrate() *ProductAutoMigrate {
	return &ProductAutoMigrate{}
}
