package entity

import "ecommerce-be/common/db"

type Inventory struct {
	db.BaseEntity
	VariantID uint `json:"variantId" gorm:"column:variant_id;not null;uniqueIndex:idx_inv_var_loc"`

	// Foreign Key to Location
	LocationID uint `json:"locationId" gorm:"column:location_id;not null;uniqueIndex:idx_inv_var_loc"`

	// Can be negative for backorder support
	Quantity         int `json:"quantity"         gorm:"column:quantity;default:0"`
	ReservedQuantity int `json:"reservedQuantity" gorm:"column:reserved_quantity;default:0;check:reserved_quantity >= 0"`
	
	// Minimum allowed quantity - can be negative for backorder limit (e.g., -10 means allow up to 10 backorders)
	Threshold int `json:"threshold" gorm:"column:threshold;default:0"`

	// Specific bin/shelf location within the warehouse (Optional)
	BinLocation string `json:"binLocation" gorm:"column:bin_location"`
}
