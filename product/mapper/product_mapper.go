package mapper

import "ecommerce-be/common/db"

type CategoryWithProductCount struct {
	CategoryID   uint   `json:"category_id"`
	CategoryName string `json:"category_name"`
	ParentID     *uint  `json:"parent_id"`
	ProductCount uint   `json:"product_count"`
}

type AttributeWithProductCount struct {
	Name          string         `json:"name"`
	Key           string         `json:"key"`
	AllowedValues db.StringArray `json:"allowed_values"`
	ProductCount  uint           `json:"product_count"`
}

type BrandWithProductCount struct {
	Brand        string `json:"brand"`
	ProductCount uint   `json:"product_count"`
}

// Variant filter mappers
type PriceRangeData struct {
	MinPrice     float64 `json:"min_price"`
	MaxPrice     float64 `json:"max_price"`
	ProductCount uint    `json:"product_count"`
}

type VariantOptionData struct {
	OptionID          uint   `json:"option_id"`
	OptionName        string `json:"option_name"`
	OptionDisplayName string `json:"option_display_name"`
	ValueID           uint   `json:"value_id"`
	OptionValue       string `json:"option_value"`
	ValueDisplayName  string `json:"value_display_name"`
	ColorCode         string `json:"color_code"`
	ProductCount      uint   `json:"product_count"`
}

type StockStatusData struct {
	InStock       uint `json:"in_stock"`
	OutOfStock    uint `json:"out_of_stock"`
	TotalProducts uint `json:"total_products"`
}
