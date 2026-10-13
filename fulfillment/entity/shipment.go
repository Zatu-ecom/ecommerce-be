package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// ============================================================================
// Shipment Status — the single vocabulary (data-model §2a).
// Statuses, ShipmentActions, and ledger event types share these strings.
// ============================================================================

type ShipmentStatus string

const (
	SHIPMENT_STATUS_DRAFT            ShipmentStatus = "draft"
	SHIPMENT_STATUS_BOOKED           ShipmentStatus = "booked"
	SHIPMENT_STATUS_PICKUP_SCHEDULED ShipmentStatus = "pickup_scheduled"
	SHIPMENT_STATUS_PICKED           ShipmentStatus = "picked"
	SHIPMENT_STATUS_IN_TRANSIT       ShipmentStatus = "in_transit"
	SHIPMENT_STATUS_OUT_FOR_DELIVERY ShipmentStatus = "out_for_delivery"
	SHIPMENT_STATUS_NDR_PENDING      ShipmentStatus = "ndr_pending"
	SHIPMENT_STATUS_DELIVERED        ShipmentStatus = "delivered"
	SHIPMENT_STATUS_FAILED           ShipmentStatus = "failed"
	SHIPMENT_STATUS_CANCELLED        ShipmentStatus = "cancelled"
	SHIPMENT_STATUS_RTO_IN_TRANSIT   ShipmentStatus = "rto_in_transit"
	SHIPMENT_STATUS_RETURNED         ShipmentStatus = "returned"
)

// ValidShipmentStatuses returns every shippable status value.
func ValidShipmentStatuses() []ShipmentStatus {
	return []ShipmentStatus{
		SHIPMENT_STATUS_DRAFT,
		SHIPMENT_STATUS_BOOKED,
		SHIPMENT_STATUS_PICKUP_SCHEDULED,
		SHIPMENT_STATUS_PICKED,
		SHIPMENT_STATUS_IN_TRANSIT,
		SHIPMENT_STATUS_OUT_FOR_DELIVERY,
		SHIPMENT_STATUS_NDR_PENDING,
		SHIPMENT_STATUS_DELIVERED,
		SHIPMENT_STATUS_FAILED,
		SHIPMENT_STATUS_CANCELLED,
		SHIPMENT_STATUS_RTO_IN_TRANSIT,
		SHIPMENT_STATUS_RETURNED,
	}
}

// TerminalShipmentStatuses holds states a box never leaves.
func TerminalShipmentStatuses() []ShipmentStatus {
	return []ShipmentStatus{
		SHIPMENT_STATUS_DELIVERED,
		SHIPMENT_STATUS_FAILED,
		SHIPMENT_STATUS_CANCELLED,
		SHIPMENT_STATUS_RETURNED,
	}
}

// String returns the string representation.
func (s ShipmentStatus) String() string {
	return string(s)
}

// IsValid checks if the shipment status is valid.
func (s ShipmentStatus) IsValid() bool {
	for _, valid := range ValidShipmentStatuses() {
		if s == valid {
			return true
		}
	}
	return false
}

// IsTerminal reports whether the status is terminal (never leaves).
func (s ShipmentStatus) IsTerminal() bool {
	for _, terminal := range TerminalShipmentStatuses() {
		if s == terminal {
			return true
		}
	}
	return false
}

// ============================================================================
// FulfillmentShipment — one row per shippable box (032, T4).
// An order split across warehouses or partial quantities is N rows.
// ============================================================================

type FulfillmentShipment struct {
	db.BaseEntity
	OrderID  uint `json:"orderId"  gorm:"column:order_id;not null;index"`
	SellerID uint `json:"sellerId" gorm:"column:seller_id;not null;index"`

	// NULL in draft (drafts are pre-provider); set at book. No default.
	ProviderCode *string `json:"providerCode" gorm:"column:provider_code;size:50"`
	// NULL until book; then the frozen config row (creds + env + flags).
	ProviderConfigID *uint `json:"providerConfigId" gorm:"column:provider_config_id"`

	// NULL in draft. Value sent to the courier is this shipment's id.
	ProviderOrderID    *string `json:"providerOrderId"    gorm:"column:provider_order_id"`
	ProviderShipmentID *string `json:"providerShipmentId" gorm:"column:provider_shipment_id"`
	// NULL until the AWB step. Webhook locator.
	AWB *string `json:"awb" gorm:"column:awb"`
	// Client retry key (Idempotency-Key header).
	IdempotencyKey *string `json:"idempotencyKey" gorm:"column:idempotency_key"`

	// Late-bound by aggregators (e.g. 'Delhivery Surface').
	CourierName *string `json:"courierName" gorm:"column:courier_name;size:100"`
	ServiceCode *string `json:"serviceCode"  gorm:"column:service_code;size:50"`

	Status ShipmentStatus `json:"status" gorm:"column:status;size:32;not null;default:draft"`

	// Logical cross-module references (NO FK): resolved via service hooks.
	PickupLocationID  *uint `json:"pickupLocationId"  gorm:"column:pickup_location_id"`
	DeliveryAddressID *uint `json:"deliveryAddressId" gorm:"column:delivery_address_id"`
	// Address revision stamp (from order_address.updated_at at plan time).
	DeliveryAddressRevisedAt *time.Time `json:"deliveryAddressRevisedAt" gorm:"column:delivery_address_revised_at"`

	WeightGrams *int     `json:"weightGrams" gorm:"column:weight_grams"`
	LengthCm    *float64 `json:"lengthCm"    gorm:"column:length_cm"`
	BreadthCm   *float64 `json:"breadthCm"   gorm:"column:breadth_cm"`
	HeightCm    *float64 `json:"heightCm"    gorm:"column:height_cm"`

	// Collect-amount snapshot. 0 = prepaid. Not a remittance record.
	CodCents int64 `json:"codCents" gorm:"column:cod_cents;not null;default:0"`
	// Quoted freight at ship time (audit).
	RateCents *int64 `json:"rateCents" gorm:"column:rate_cents"`
	Insured   bool   `json:"insured"   gorm:"column:insured;not null;default:false"`

	ETD             *time.Time `json:"etd"           gorm:"column:etd"`
	ShippedAt       *time.Time `json:"shippedAt"     gorm:"column:shipped_at"`
	DeliveredAt     *time.Time `json:"deliveredAt"   gorm:"column:delivered_at"`
	CancelledAt     *time.Time `json:"cancelledAt"   gorm:"column:cancelled_at"`
	LastSyncedAt    *time.Time `json:"lastSyncedAt"  gorm:"column:last_synced_at"`
	BookRequestedAt *time.Time `json:"bookRequestedAt" gorm:"column:book_requested_at"`
	BookAttempts    int        `json:"bookAttempts"  gorm:"column:book_attempts;not null;default:0"`

	// Return boxes point at the original. No separate return-status table.
	ReturnOfShipmentID *uint `json:"returnOfShipmentId" gorm:"column:return_of_shipment_id"`

	// Sanitized provider snapshot, no PII.
	RawRef db.JSONMap `json:"rawRef" gorm:"column:raw_ref;type:jsonb;not null;default:'{}'"`

	// Relationships (Preload to avoid N+1).
	Items []FulfillmentShipmentItem `json:"items,omitempty" gorm:"foreignKey:ShipmentID"`
}

// TableName returns the singular table name.
func (FulfillmentShipment) TableName() string {
	return "fulfillment_shipment"
}

// ============================================================================
// FulfillmentShipmentItem — split lines (032, T5).
// ============================================================================

type FulfillmentShipmentItem struct {
	db.BaseEntity
	ShipmentID  uint `json:"shipmentId"  gorm:"column:shipment_id;not null;index"`
	OrderItemID uint `json:"orderItemId" gorm:"column:order_item_id;not null;index"`
	Quantity    int  `json:"quantity"    gorm:"column:quantity;not null"`
}

// TableName returns the singular table name.
func (FulfillmentShipmentItem) TableName() string {
	return "fulfillment_shipment_item"
}
