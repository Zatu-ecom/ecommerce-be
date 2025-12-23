package entity

import (
	"fmt"
	"strings"

	"ecommerce-be/common/db"

	"gorm.io/gorm"
)

type LocationType string

const (
	LOC_WAREHOUSE     LocationType = "WAREHOUSE"
	LOC_STORE         LocationType = "STORE"
	LOC_RETURN_CENTER LocationType = "RETURN_CENTER"
)

// ValidLocationTypes returns all valid location type values
func ValidLocationTypes() []LocationType {
	return []LocationType{
		LOC_WAREHOUSE,
		LOC_STORE,
		LOC_RETURN_CENTER,
	}
}

// String returns the string representation
func (lt LocationType) String() string {
	return string(lt)
}

// IsValid checks if the location type is valid
func (lt LocationType) IsValid() bool {
	switch lt {
	case LOC_WAREHOUSE, LOC_STORE, LOC_RETURN_CENTER:
		return true
	}
	return false
}

// Location represents a physical place where inventory is stored
type Location struct {
	db.BaseEntity

	// Basic Info
	Name     string       `json:"name"     gorm:"column:name;size:255;not null"`
	Type     LocationType `json:"type"     gorm:"column:type;size:20;default:WAREHOUSE"`
	IsActive bool         `json:"isActive" gorm:"column:is_active;default:true"`
	Priority int          `json:"priority" gorm:"column:priority;default:0"` // Higher number = Higher priority (or lower, depending on logic. Usually 1 is highest)

	// Multi-tenant
	SellerID uint `json:"sellerId" gorm:"column:seller_id;not null;index"`

	// Address Reference
	AddressID uint `json:"addressId" gorm:"column:address_id;not null"`
}

// BeforeSave GORM hook - validates LocationType before insert
func (l *Location) BeforeSave(tx *gorm.DB) error {
	if !l.Type.IsValid() {
		validTypes := ValidLocationTypes()
		typeStrings := make([]string, len(validTypes))
		for i, t := range validTypes {
			typeStrings[i] = string(t)
		}
		return fmt.Errorf(
			"invalid location type: must be one of %s",
			strings.Join(typeStrings, ", "),
		)
	}
	return nil
}

// BeforeUpdate GORM hook - validates LocationType before update
func (l *Location) BeforeUpdate(tx *gorm.DB) error {
	if !l.Type.IsValid() {
		validTypes := ValidLocationTypes()
		typeStrings := make([]string, len(validTypes))
		for i, t := range validTypes {
			typeStrings[i] = string(t)
		}
		return fmt.Errorf(
			"invalid location type: must be one of %s",
			strings.Join(typeStrings, ", "),
		)
	}
	return nil
}
