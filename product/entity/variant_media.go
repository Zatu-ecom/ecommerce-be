package entity

import "ecommerce-be/common/db"

// VariantMedia represents a file asset associated with a product variant.
// All binary assets are owned by the File module; this table holds only the
// association metadata (ordering and primary flag).
type VariantMedia struct {
	db.BaseEntity
	VariantID    uint   `json:"variantId"    gorm:"column:variant_id;not null"`
	FileID       string `json:"fileId"       gorm:"column:file_id;not null"`
	IsPrimary    bool   `json:"isPrimary"    gorm:"column:is_primary;default:false"`
	DisplayOrder int    `json:"displayOrder" gorm:"column:display_order;default:0"`
}
