package mapper

import "ecommerce-be/common/db"

type CategoryWithProductCount struct {
	CategoryID   uint   `json:"category_id"`
	CategoryName string `json:"category_name"`
	ParentID     *uint  `json:"parent_id"`
	ProductCount uint   `json:"product_count"`
}

type AttributeWithProductCount struct {
	Name          string             `json:"name"`
	Key           string             `json:"key"`
	AllowedValues db.StringArray `json:"allowed_values"`
	ProductCount  uint               `json:"product_count"`
}

type BrandWithProductCount struct {
	Brand        string `json:"brand"`
	ProductCount uint   `json:"product_count"`
}
