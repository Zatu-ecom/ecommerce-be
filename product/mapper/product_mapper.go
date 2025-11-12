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

// RelatedProductScored represents the scored related product data from stored procedure
type RelatedProductScored struct {
	ProductID          uint           `gorm:"column:product_id"`
	ProductName        string         `gorm:"column:product_name"`
	CategoryID         uint           `gorm:"column:category_id"`
	CategoryName       string         `gorm:"column:category_name"`
	ParentCategoryID   *uint          `gorm:"column:parent_category_id"`
	ParentCategoryName *string        `gorm:"column:parent_category_name"`
	Brand              string         `gorm:"column:brand"`
	SKU                string         `gorm:"column:sku"`
	ShortDescription   string         `gorm:"column:short_description"`
	LongDescription    string         `gorm:"column:long_description"`
	Tags               db.StringArray `gorm:"column:tags;type:text[]"`
	SellerID           uint           `gorm:"column:seller_id"`
	HasVariants        bool           `gorm:"column:has_variants"`
	MinPrice           float64        `gorm:"column:min_price"`
	MaxPrice           float64        `gorm:"column:max_price"`
	AllowPurchase      bool           `gorm:"column:allow_purchase"`
	TotalVariants      int64          `gorm:"column:total_variants"`
	InStockVariants    int64          `gorm:"column:in_stock_variants"`
	CreatedAt          string         `gorm:"column:created_at"`
	UpdatedAt          string         `gorm:"column:updated_at"`
	FinalScore         int            `gorm:"column:final_score"`
	RelationReason     string         `gorm:"column:relation_reason"`
	StrategyUsed       string         `gorm:"column:strategy_used"`
}
