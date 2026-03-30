package entity

import (
	"ecommerce-be/common/db"
)

// CustomerSegment represents a dynamic group of customers based on rules
type CustomerSegment struct {
	db.BaseEntity

	// Owner
	SellerID uint `json:"sellerId" gorm:"column:seller_id;not null;index"`

	// Info
	Name        string  `json:"name"        gorm:"column:name;size:255;not null"`
	Description *string `json:"description" gorm:"column:description;type:text"`

	// Rules for segmentation
	// Example: {"operator": "AND", "conditions": [{"field": "total_spent", "op": ">", "value": 1000}]}
	Rules SegmentRules `json:"rules" gorm:"column:rules;type:jsonb;not null"`

	// Metadata
	Metadata db.JSONMap `json:"metadata" gorm:"column:metadata;type:jsonb;default:'{}'"`
}

// SegmentRules holds the segmentation logic in JSONB format.
type SegmentRules = db.JSONMap

// TableName specifies the table name
func (CustomerSegment) TableName() string {
	return "customer_segment"
}
