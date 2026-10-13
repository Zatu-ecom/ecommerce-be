package entity

import (
	"time"
)

// PhysicalSpecUnit is one allowed unit for one shippable measurement
// (migration 033, core seed 005). Keys double as attribute_definition keys:
// the seller's "unit choice" is which definition row they attach, so
// product_attribute needs no new column.
type PhysicalSpecUnit struct {
	Key                   string  `json:"key"                   gorm:"column:key;primaryKey;size:50"`
	AttributeDefinitionID uint    `json:"attributeDefinitionId" gorm:"column:attribute_definition_id;not null;uniqueIndex"`
	Parameter             string  `json:"parameter"             gorm:"column:parameter;size:20;not null;index"`
	UnitShort             string  `json:"unitShort"             gorm:"column:unit_short;size:10;not null"`
	UnitFull              string  `json:"unitFull"              gorm:"column:unit_full;size:50;not null"`
	FactorToBase          float64 `json:"factorToBase"          gorm:"column:factor_to_base;not null"`
	IsBase                bool    `json:"isBase"                gorm:"column:is_base;not null;default:false"`
	DisplayOrder          int     `json:"displayOrder"          gorm:"column:display_order;not null;default:0"`
	IsActive              bool    `json:"isActive"              gorm:"column:is_active;not null;default:true"`

	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the singular table name.
func (PhysicalSpecUnit) TableName() string {
	return "physical_spec_unit"
}
