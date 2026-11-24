package model

import (
	"time"
)

// ─── Courier DTOs (provider-agnostic) ───────────────────────────────────────
// Location rule: pincodes and pickup nicknames are resolved in memory from
// pickup_location_id / delivery_address_id via hooks at call time. The
// shipment persists ids only.

// RateInput is the provider-agnostic input for shopping live rates.
// Pincodes arrive hook-resolved (availability rows carry warehouse pincodes;
// the order view carries the delivery pincode) — never stored columns.
// CurrencyCode is the ISO display currency (e.g. INR); adapters MUST use it
// for minor<->major conversion via common/model (never hardcoded *100).
type RateInput struct {
	PickupPincode   string
	DeliveryPincode string
	WeightGrams     int
	LengthCm        float64
	BreadthCm       float64
	HeightCm        float64
	CodCents        int64
	OrderValueCents int64
	// CurrencyCode drives currency-aware conversion (default INR when empty).
	CurrencyCode string
}

// RateOption is one priced courier service for a rate request.
type RateOption struct {
	CourierName   string
	ServiceCode   string
	RateCents     int64
	ETD           *time.Time
	PickupCapable bool
}

// ShipmentItemInput is one order line packed into a box.
type ShipmentItemInput struct {
	OrderItemID uint
	Quantity    int
}

// BookShipmentInput is the provider-agnostic input for booking a draft.
// PickupLocationID lets the adapter derive the pickup nickname
// (S{seller}L{location}) at book time. ShipTo + Lines carry everything the
// courier manifest needs; the service resolves them from the order view.
// All money stays integer minor units; CurrencyCode (ISO, e.g. INR) tells
// adapters how to render major units for provider payloads via common/model.
type BookShipmentInput struct {
	ShipmentID       uint
	OrderID          uint
	SellerID         uint
	Items            []ShipmentItemInput
	PickupLocationID uint
	PickupAt         *time.Time
	WeightGrams      *int
	LengthCm         *float64
	BreadthCm        *float64
	HeightCm         *float64
	CourierHint      string
	RecipientName    string
	RecipientPhone   string
	Street           string
	City             string
	State            string
	Pincode          string
	Lines            []BookLine
	CodCents         int64
	SubTotalCents    int64
	// CurrencyCode is the order display currency (fallback INR when empty).
	CurrencyCode string
}

// BookLine is one manifest line for the courier payload.
type BookLine struct {
	Name           string
	SKU            string
	Quantity       int
	UnitPriceCents int64
}

// BookShipmentOutput is the provider-agnostic result of booking.
type BookShipmentOutput struct {
	ProviderOrderID    string
	ProviderShipmentID string
	AWB                string
	CourierName        string
	ServiceCode        string
	ETD                *time.Time
	RawResponse        map[string]any
}

// PickupInput schedules pickup separately from booking (optional capability).
type PickupInput struct {
	ShipmentID         uint
	AWB                string
	ProviderShipmentID string
	PickupAt           *time.Time
}

// CancelInput cancels a pre-pickup booking.
type CancelInput struct {
	ShipmentID      uint
	AWB             string
	ProviderOrderID string
}

// LabelInput fetches the printable label bytes (streamed, never stored).
type LabelInput struct {
	ShipmentID         uint
	AWB                string
	ProviderShipmentID string
}

// NDRDetail is one failed-delivery round as the provider reports it.
type NDRDetail struct {
	AWB       string
	NDRStatus string
	Reason    string
	AttemptNo int
	Raw       map[string]any
}

// NDRActionInput answers an open NDR round: reattempt or RTO.
type NDRActionInput struct {
	AWB         string
	Action      string // 'reattempt', 'rto'
	AddressNote string
}

// BookReturnInput books a return box linked to its original shipment.
// Ship carries the full shipping context (composed by the service exactly
// like a forward book: recipient, manifest lines, weight); the adapter adds
// the return marker and original reference.
type BookReturnInput struct {
	OrigShipmentID uint
	Reason         string
	Items          []ShipmentItemInput
	Ship           BookShipmentInput
}

// RTORequestInput is a proactive mid-transit return request (no open NDR).
type RTORequestInput struct {
	ShipmentID uint
	AWB        string
	Reason     string
}
