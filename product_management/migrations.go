package product_management

import (
	"ecommerce-be/product_management/entity"
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
