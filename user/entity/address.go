package entity

import (
	"fmt"
	"strings"

	"ecommerce-be/common/db"

	"gorm.io/gorm"
)

// AddressType represents the type/purpose of an address
type AddressType string

const (
	ADDR_HOME          AddressType = "HOME"
	ADDR_WORK          AddressType = "WORK"
	ADDR_WAREHOUSE     AddressType = "WAREHOUSE"
	ADDR_STORE         AddressType = "STORE"
	ADDR_RETURN_CENTER AddressType = "RETURN_CENTER"
	ADDR_OTHER         AddressType = "OTHER"
)

// ValidAddressTypes returns all valid address type values
func ValidAddressTypes() []AddressType {
	return []AddressType{
		ADDR_HOME,
		ADDR_WORK,
		ADDR_WAREHOUSE,
		ADDR_STORE,
		ADDR_RETURN_CENTER,
		ADDR_OTHER,
	}
}

// String returns the string representation
func (at AddressType) String() string {
	return string(at)
}

// IsValid checks if the address type is valid
func (at AddressType) IsValid() bool {
	switch at {
	case ADDR_HOME, ADDR_WORK, ADDR_WAREHOUSE, ADDR_STORE, ADDR_RETURN_CENTER, ADDR_OTHER:
		return true
	}
	return false
}

// IsLocationAddress checks if this address type is for a location (warehouse, store, etc.)
func (at AddressType) IsLocationAddress() bool {
	switch at {
	case ADDR_WAREHOUSE, ADDR_STORE, ADDR_RETURN_CENTER:
		return true
	}
	return false
}

type Address struct {
	db.BaseEntity
	UserID    uint        `json:"userId"    gorm:"column:user_id;not null;index"`
	Type      AddressType `json:"type"      gorm:"column:type;size:20;default:HOME"`
	Address   string      `json:"address"   gorm:"column:address;size:500;not null"`
	Landmark  string      `json:"landmark"  gorm:"column:landmark;size:255"`
	City      string      `json:"city"      gorm:"column:city;size:100;not null"`
	State     string      `json:"state"     gorm:"column:state;size:100;not null"`
	ZipCode   string      `json:"zipCode"   gorm:"column:zip_code;size:20;not null"`
	CountryID uint        `json:"countryId" gorm:"column:country_id;not null"`
	Latitude  *float64    `json:"latitude"  gorm:"column:latitude"`
	Longitude *float64    `json:"longitude" gorm:"column:longitude"`
	IsDefault bool        `json:"isDefault" gorm:"column:is_default;default:false"`

	// Relationships
	Country Country `json:"country,omitempty" gorm:"foreignKey:CountryID"`
}

// BeforeSave GORM hook - validates AddressType before insert
func (a *Address) BeforeSave(tx *gorm.DB) error {
	if !a.Type.IsValid() {
		validTypes := ValidAddressTypes()
		typeStrings := make([]string, len(validTypes))
		for i, t := range validTypes {
			typeStrings[i] = string(t)
		}
		return fmt.Errorf(
			"invalid address type: must be one of %s",
			strings.Join(typeStrings, ", "),
		)
	}
	return nil
}

// BeforeUpdate GORM hook - validates AddressType before update
func (a *Address) BeforeUpdate(tx *gorm.DB) error {
	if !a.Type.IsValid() {
		validTypes := ValidAddressTypes()
		typeStrings := make([]string, len(validTypes))
		for i, t := range validTypes {
			typeStrings[i] = string(t)
		}
		return fmt.Errorf(
			"invalid address type: must be one of %s",
			strings.Join(typeStrings, ", "),
		)
	}
	return nil
}
