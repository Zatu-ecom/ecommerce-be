package entity

import (
	"database/sql/driver"
	"encoding/json"

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
	Metadata JSONMap `json:"metadata" gorm:"column:metadata;type:jsonb;default:'{}'"`
}

// SegmentRules holds the segmentation logic in JSONB format
type SegmentRules map[string]interface{}

// Scan implements sql.Scanner interface
func (sr *SegmentRules) Scan(value interface{}) error {
	if value == nil {
		*sr = make(SegmentRules)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, sr)
}

// Value implements driver.Valuer interface
func (sr SegmentRules) Value() (driver.Value, error) {
	if sr == nil {
		return json.Marshal(make(map[string]interface{}))
	}
	return json.Marshal(sr)
}

// TableName specifies the table name
func (CustomerSegment) TableName() string {
	return "customer_segment"
}
