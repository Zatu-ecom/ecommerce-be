package mapper

import "ecommerce-be/common/entity"

type CategoryWithProductCount struct {
	CategoryID   uint   `json:"category_id"`
	CategoryName string `json:"category_name"`
	ParentID     *uint  `json:"parent_id"`
	ProductCount uint   `json:"product_count"`
}

type AttributeWithProductCount struct {
	Name          string             `json:"name"`
	Key           string             `json:"key"`
	AllowedValues entity.StringArray `json:"allowed_values"`
	ProductCount  uint               `json:"product_count"`
}

type BrandWithProductCount struct {
	Brand        string `json:"brand"`
	ProductCount uint   `json:"product_count"`
}
