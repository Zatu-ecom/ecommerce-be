package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// ============================================================================
// Order Shipment Status Enum
// ============================================================================

type ShipmentStatus string

const (
	SHIPMENT_STATUS_PENDING          ShipmentStatus = "pending"
	SHIPMENT_STATUS_PICKED           ShipmentStatus = "picked"
	SHIPMENT_STATUS_IN_TRANSIT       ShipmentStatus = "in_transit"
	SHIPMENT_STATUS_OUT_FOR_DELIVERY ShipmentStatus = "out_for_delivery"
	SHIPMENT_STATUS_DELIVERED        ShipmentStatus = "delivered"
	SHIPMENT_STATUS_FAILED           ShipmentStatus = "failed"
	SHIPMENT_STATUS_RETURNED         ShipmentStatus = "returned"
)

// ValidShipmentStatuses returns all valid shipment status values
func ValidShipmentStatuses() []ShipmentStatus {
	return []ShipmentStatus{
		SHIPMENT_STATUS_PENDING,
		SHIPMENT_STATUS_PICKED,
		SHIPMENT_STATUS_IN_TRANSIT,
		SHIPMENT_STATUS_OUT_FOR_DELIVERY,
		SHIPMENT_STATUS_DELIVERED,
		SHIPMENT_STATUS_FAILED,
		SHIPMENT_STATUS_RETURNED,
	}
}

// String returns the string representation
func (s ShipmentStatus) String() string {
	return string(s)
}

// IsValid checks if the shipment status is valid
func (s ShipmentStatus) IsValid() bool {
	switch s {
	case SHIPMENT_STATUS_PENDING, SHIPMENT_STATUS_PICKED, SHIPMENT_STATUS_IN_TRANSIT,
		SHIPMENT_STATUS_OUT_FOR_DELIVERY, SHIPMENT_STATUS_DELIVERED,
		SHIPMENT_STATUS_FAILED, SHIPMENT_STATUS_RETURNED:
		return true
	}
	return false
}

// ============================================================================
// Order Shipment Entity
// ============================================================================

type OrderShipment struct {
	db.BaseEntity
	OrderID     uint                   `json:"orderId"     gorm:"column:order_id;not null;index"`
	Carrier     string                 `json:"carrier"     gorm:"column:carrier;size:50"`
	TrackingNo  string                 `json:"trackingNo"  gorm:"column:tracking_no;size:100"`
	Status      ShipmentStatus         `json:"status"      gorm:"column:status;size:32;default:pending;index"`
	ShippedAt   *time.Time             `json:"shippedAt"   gorm:"column:shipped_at"`
	DeliveredAt *time.Time             `json:"deliveredAt" gorm:"column:delivered_at"`
	Metadata    map[string]interface{} `json:"metadata"    gorm:"column:metadata;type:jsonb;default:'{}'"`

	// Relationships
	Items []OrderShipmentItem `json:"items,omitempty" gorm:"foreignKey:ShipmentID"`
}

// ============================================================================
// Order Shipment Item Entity
// ============================================================================

// OrderShipmentItem links shipments to order items
// Supports partial shipments (splitting one order item across multiple shipments)
type OrderShipmentItem struct {
	db.BaseEntity
	ShipmentID  uint `json:"shipmentId"  gorm:"column:shipment_id;not null;index"`
	OrderItemID uint `json:"orderItemId" gorm:"column:order_item_id;not null;index"`
	Quantity    int  `json:"quantity"    gorm:"column:quantity;not null"` // Quantity in this shipment (can be partial)
}
