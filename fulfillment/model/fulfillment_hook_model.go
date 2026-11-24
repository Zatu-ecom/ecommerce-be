package model

import (
	"time"
)

// ─── Cross-module hook DTOs ─────────────────────────────────────────────────
// These types live in the leaf model package (no service imports) so the
// order and inventory modules can implement fulfillment hook interfaces
// without creating import cycles. fulfillment/service/hooks.go owns the
// interfaces; this file owns the shared shapes.

// FulfillmentOrderView is the planning input for one order.
type FulfillmentOrderView struct {
	OrderID         uint
	SellerID        uint
	UserID          uint
	Items           []FulfillmentOrderItemView
	FulfillmentType string
	// Status gates planning: only confirmed orders plan (pending → 409).
	Status string
	// DeliveryAddressID is the logical ref to order_address; pincode and
	// revision timestamp travel in memory only, never stored.
	DeliveryAddressID        uint
	DeliveryAddressUpdatedAt time.Time
	DeliveryPincode          string
	// Delivery street/city/state + recipient come from the order address
	// snapshot and the customer profile: courier payloads need them, and
	// fulfillment must not re-read user tables itself.
	DeliveryStreet string
	DeliveryCity   string
	DeliveryState  string
	RecipientName  string
	RecipientPhone string
	// CodCents is the collect amount (0 = prepaid).
	CodCents int64
	// CurrencyCode is the seller's display currency (ISO code). Empty when
	// unresolvable — callers fall back to INR display, never block planning
	// on a display-only concern.
	CurrencyCode string
}

// FulfillmentOrderItemView is one shippable order line. Names and prices
// travel along so courier payloads (manifests, customs) render without a
// second order read at book time.
type FulfillmentOrderItemView struct {
	OrderItemID    uint
	VariantID      *uint
	Quantity       int
	ProductName    string
	SKU            string
	UnitPriceCents int64
}

// FulfillmentLine is one box line: order item id + quantity in this box.
type FulfillmentLine struct {
	OrderItemID uint
	Quantity    int
}

// FulfillmentProgress is the per-box event handed to order hooks.
// Reason carries the failure/return why (pinned vocabularies in
// service-design §6: cancelled_pre_pickup | lost | damaged).
// PickupLocationID tells stock moves which warehouse the box left from.
type FulfillmentProgress struct {
	OrderID          uint
	ShipmentID       uint
	Items            []FulfillmentLine
	Reason           string
	PickupLocationID uint
}

// AvailabilityRow is one variant's availability at one warehouse.
// AvailableQty excludes the caller's own holds; HeldQty is this order's
// CONFIRMED hold at the same spot. Planners allocate from their sum.
type AvailabilityRow struct {
	VariantID    uint
	LocationID   uint
	AvailableQty int
	HeldQty      int
	Priority     int
	Pincode      string
	SellerID     uint
}

// ReservationLine addresses held units for adopt/release. OrderID locates
// the CONFIRMED hold (reservations are keyed by order reference).
// LocationID 0 means "any location holding this variant for the order"
// (used when the caller knows variants but not warehouses); otherwise the
// match is exact.
type ReservationLine struct {
	OrderID    uint
	VariantID  *uint
	LocationID uint
	Quantity   int
	ShipmentID uint
}

// RestockLine returns previously shipped units to stock. The warehouse is
// resolved from the consumed FULFILLED rows themselves, so no location is
// needed — variants that never shipped for the order restock nothing.
type RestockLine struct {
	VariantID *uint
	Quantity  int
}

// PhysicalSpec is one variant's shippable measurements in base units
// (grams, cm). Absent fields mean the product carries no such spec.
type PhysicalSpec struct {
	VariantID   uint
	WeightGrams *int
	LengthCm    *float64
	BreadthCm   *float64
	HeightCm    *float64
}
