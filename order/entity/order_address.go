package entity

import "ecommerce-be/common/db"

// ============================================================================
// Order Address Type Enum
// ============================================================================

type OrderAddressType string

const (
	ORDER_ADDR_SHIPPING OrderAddressType = "shipping"
	ORDER_ADDR_BILLING  OrderAddressType = "billing"
)

// ValidOrderAddressTypes returns all valid order address type values
func ValidOrderAddressTypes() []OrderAddressType {
	return []OrderAddressType{
		ORDER_ADDR_SHIPPING,
		ORDER_ADDR_BILLING,
	}
}

// String returns the string representation
func (t OrderAddressType) String() string {
	return string(t)
}

// IsValid checks if the order address type is valid
func (t OrderAddressType) IsValid() bool {
	switch t {
	case ORDER_ADDR_SHIPPING, ORDER_ADDR_BILLING:
		return true
	}
	return false
}

// ============================================================================
// Order Address Entity
// ============================================================================

// OrderAddress stores a snapshot of address at time of order placement
// This ensures order history is preserved even if user changes/deletes their address
type OrderAddress struct {
	db.BaseEntity
	OrderID   uint             `json:"orderId"   gorm:"column:order_id;not null;index"`
	Type      OrderAddressType `json:"type"      gorm:"column:type;size:20;not null"`
	Address   string           `json:"address"   gorm:"column:address;size:500;not null"`
	Landmark  string           `json:"landmark"  gorm:"column:landmark;size:255"`
	City      string           `json:"city"      gorm:"column:city;size:100;not null"`
	State     string           `json:"state"     gorm:"column:state;size:100;not null"`
	ZipCode   string           `json:"zipCode"   gorm:"column:zip_code;size:20;not null"`
	CountryID uint             `json:"countryId" gorm:"column:country_id;not null"`
	Latitude  *float64         `json:"latitude"  gorm:"column:latitude"`
	Longitude *float64         `json:"longitude" gorm:"column:longitude"`
}
