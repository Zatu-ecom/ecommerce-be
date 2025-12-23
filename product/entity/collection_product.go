package entity

import (
	"ecommerce-be/common/db"
)

// CollectionProduct represents the many-to-many relationship between collections and products
type CollectionProduct struct {
	db.BaseEntity

	// References
	CollectionID uint `json:"collectionId" gorm:"column:collection_id;not null;index"`
	ProductID    uint `json:"productId"    gorm:"column:product_id;not null;index"`

	// Display order within the collection
	Position int `json:"position" gorm:"column:position;default:0"`

	// Relationships
	Collection *Collection `json:"collection,omitempty" gorm:"foreignKey:CollectionID"`
	Product    *Product    `json:"product,omitempty"    gorm:"foreignKey:ProductID"`
}

// TableName specifies the table name
func (CollectionProduct) TableName() string {
	return "collection_product"
}
