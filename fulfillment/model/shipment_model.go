package model

import (
	"strings"
	"time"

	commonModel "ecommerce-be/common/model"
	"ecommerce-be/fulfillment/entity"
)

// ─── Shipment DTOs (api-contracts.md §§1–2, 4) ───────────────────────────────
// Times serialize RFC3339 (encoding/json default for time.Time). Ids are
// numbers. pickupLocationId/deliveryAddressId stay ids; the client loads
// addresses from the order API.

// CreateShipmentRequest is a manual draft for lines the planner did not cover.
type CreateShipmentRequest struct {
	OrderID          uint                  `json:"orderId" binding:"required"`
	PickupLocationID uint                  `json:"pickupLocationId" binding:"required"`
	Items            []ShipmentItemRequest `json:"items" binding:"required,min=1,dive"`
	WeightGrams      *int                  `json:"weightGrams" binding:"omitempty,gt=0"`
	LengthCm         *float64              `json:"lengthCm" binding:"omitempty,gt=0"`
	BreadthCm        *float64              `json:"breadthCm" binding:"omitempty,gt=0"`
	HeightCm         *float64              `json:"heightCm" binding:"omitempty,gt=0"`
}

// ShipmentItemRequest is one order line packed into the draft.
type ShipmentItemRequest struct {
	OrderItemID uint `json:"orderItemId" binding:"required"`
	Quantity    int  `json:"quantity" binding:"required,gt=0"`
}

// UpdateShipmentRequest patches draft measurements. At least one field is
// required (enforced in the service: struct tags cannot express it).
type UpdateShipmentRequest struct {
	WeightGrams *int     `json:"weightGrams" binding:"omitempty,gt=0"`
	LengthCm    *float64 `json:"lengthCm" binding:"omitempty,gt=0"`
	BreadthCm   *float64 `json:"breadthCm" binding:"omitempty,gt=0"`
	HeightCm    *float64 `json:"heightCm" binding:"omitempty,gt=0"`
}

// IsEmpty reports whether no field was sent.
func (r UpdateShipmentRequest) IsEmpty() bool {
	return r.WeightGrams == nil && r.LengthCm == nil &&
		r.BreadthCm == nil && r.HeightCm == nil
}

// ShipmentItemResponse is one box line.
type ShipmentItemResponse struct {
	OrderItemID uint `json:"orderItemId"`
	Quantity    int  `json:"quantity"`
}

// ShipmentResponse is the full box shape (detail, plan, book responses).
// List rows reuse it WITH items and WITHOUT events/ndr (items are tiny and
// save clients N+1 detail calls; events/ndr stay detail-only).
type ShipmentResponse struct {
	ID                 uint                    `json:"id"`
	OrderID            uint                    `json:"orderId"`
	SellerID           uint                    `json:"sellerId"`
	Status             string                  `json:"status"`
	ProviderCode       *string                 `json:"providerCode"`
	AWB                *string                 `json:"awb"`
	CourierName        *string                 `json:"courierName"`
	ServiceCode        *string                 `json:"serviceCode"`
	PickupLocationID   *uint                   `json:"pickupLocationId"`
	DeliveryAddressID  *uint                   `json:"deliveryAddressId"`
	WeightGrams        *int                    `json:"weightGrams"`
	LengthCm           *float64                `json:"lengthCm"`
	BreadthCm          *float64                `json:"breadthCm"`
	HeightCm           *float64                `json:"heightCm"`
	COD                commonModel.Money       `json:"cod"`
	Rate               *commonModel.Money      `json:"rate"`
	Insured            bool                    `json:"insured"`
	ETD                *time.Time              `json:"etd"`
	ShippedAt          *time.Time              `json:"shippedAt"`
	DeliveredAt        *time.Time              `json:"deliveredAt"`
	CancelledAt        *time.Time              `json:"cancelledAt"`
	LastSyncedAt       *time.Time              `json:"lastSyncedAt"`
	ReturnOfShipmentID *uint                   `json:"returnOfShipmentId"`
	BookRequestedAt    *time.Time              `json:"bookRequestedAt"`
	BookAttempts       int                     `json:"bookAttempts"`
	Items              []ShipmentItemResponse  `json:"items,omitempty"`
	Events             []ShipmentEventResponse `json:"events,omitempty"`
	NDR                *NDRRoundResponse       `json:"ndr,omitempty"`
	CreatedAt          time.Time               `json:"createdAt"`
	UpdatedAt          time.Time               `json:"updatedAt"`
}

// ShipmentEventResponse is one ledger row (seller-visible subset).
// Failure fields omit when empty so customer shapes carry no trace of them.
type ShipmentEventResponse struct {
	EventType      string    `json:"eventType"`
	FromStatus     *string   `json:"fromStatus,omitempty"`
	ToStatus       string    `json:"toStatus"`
	ProviderEvent  *string   `json:"providerEvent,omitempty"`
	Source         string    `json:"source"`
	FailureCode    *string   `json:"failureCode,omitempty"`
	FailureMessage *string   `json:"failureMessage,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

// NDRRoundResponse is the open NDR round, if any.
type NDRRoundResponse struct {
	AttemptNo   int        `json:"attemptNo"`
	NDRStatus   string     `json:"ndrStatus"`
	Reason      *string    `json:"reason"`
	ActionTaken *string    `json:"actionTaken"`
	ActedAt     *time.Time `json:"actedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// BookFailure is one per-draft failure inside a batch book response.
type BookFailure struct {
	ShipmentID uint   `json:"shipmentId"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

// NDRActionRequest answers the open NDR round: reattempt or convert to RTO.
type NDRActionRequest struct {
	Action      string `json:"action" binding:"required,oneof=reattempt rto"`
	AddressNote string `json:"addressNote"`
}

// ReturnRequest creates a return box for a delivered original.
type ReturnRequest struct {
	Reason string                `json:"reason" binding:"required"`
	Items  []ShipmentItemRequest `json:"items" binding:"required,min=1,dive"`
}

// PlanResult is the planner outcome. CreatedNew distinguishes 201 (new
// drafts) from 200 (already covered) for the plan endpoint.
type PlanResult struct {
	ShipmentIDs []uint
	CreatedNew  bool
}

// CurrencyFor resolves display currency metadata for an ISO code. Unknown
// codes fall back to a code-prefixed 2-decimal rendering (never blocks).
func CurrencyFor(code string) commonModel.CurrencyInfo {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "", "INR":
		return commonModel.CurrencyInfo{Code: "INR", Symbol: "₹", DecimalDigits: 2}
	case "USD":
		return commonModel.CurrencyInfo{Code: "USD", Symbol: "$", DecimalDigits: 2}
	default:
		upper := strings.ToUpper(strings.TrimSpace(code))
		return commonModel.CurrencyInfo{Code: upper, Symbol: upper + " ", DecimalDigits: 2}
	}
}

// Money builds the shared Money object for a cents amount.
func Money(cents int64, currencyCode string) commonModel.Money {
	return commonModel.NewMoney(cents, CurrencyFor(currencyCode))
}

// ToShipmentResponse maps a shipment + its items to the API shape.
// CurrencyCode comes from the order view (seller display currency).
func ToShipmentResponse(
	shipment *entity.FulfillmentShipment,
	items []entity.FulfillmentShipmentItem,
	currencyCode string,
) ShipmentResponse {
	resp := ShipmentResponse{
		ID:                 shipment.ID,
		OrderID:            shipment.OrderID,
		SellerID:           shipment.SellerID,
		Status:             shipment.Status.String(),
		ProviderCode:       shipment.ProviderCode,
		AWB:                shipment.AWB,
		CourierName:        shipment.CourierName,
		ServiceCode:        shipment.ServiceCode,
		PickupLocationID:   shipment.PickupLocationID,
		DeliveryAddressID:  shipment.DeliveryAddressID,
		WeightGrams:        shipment.WeightGrams,
		LengthCm:           shipment.LengthCm,
		BreadthCm:          shipment.BreadthCm,
		HeightCm:           shipment.HeightCm,
		COD:                Money(shipment.CodCents, currencyCode),
		Insured:            shipment.Insured,
		ETD:                shipment.ETD,
		ShippedAt:          shipment.ShippedAt,
		DeliveredAt:        shipment.DeliveredAt,
		CancelledAt:        shipment.CancelledAt,
		LastSyncedAt:       shipment.LastSyncedAt,
		ReturnOfShipmentID: shipment.ReturnOfShipmentID,
		BookRequestedAt:    shipment.BookRequestedAt,
		BookAttempts:       shipment.BookAttempts,
		CreatedAt:          shipment.CreatedAt,
		UpdatedAt:          shipment.UpdatedAt,
	}
	if shipment.RateCents != nil {
		rate := Money(*shipment.RateCents, currencyCode)
		resp.Rate = &rate
	}
	resp.Items = make([]ShipmentItemResponse, 0, len(items))
	for _, item := range items {
		resp.Items = append(resp.Items, ShipmentItemResponse{
			OrderItemID: item.OrderItemID,
			Quantity:    item.Quantity,
		})
	}
	return resp
}

// ToShipmentEventResponse maps a ledger row to the seller-visible subset.
func ToShipmentEventResponse(event entity.FulfillmentShipmentEvent) ShipmentEventResponse {
	return ShipmentEventResponse{
		EventType:      event.EventType,
		FromStatus:     event.FromStatus,
		ToStatus:       event.ToStatus,
		ProviderEvent:  event.ProviderEvent,
		Source:         event.Source,
		FailureCode:    event.FailureCode,
		FailureMessage: event.FailureMessage,
		CreatedAt:      event.CreatedAt,
	}
}

// ToNDRRoundResponse maps the open NDR round (nil when none).
func ToNDRRoundResponse(round *entity.FulfillmentNDR) *NDRRoundResponse {
	if round == nil {
		return nil
	}
	return &NDRRoundResponse{
		AttemptNo:   round.AttemptNo,
		NDRStatus:   round.NDRStatus,
		Reason:      round.Reason,
		ActionTaken: round.ActionTaken,
		ActedAt:     round.ActedAt,
		CreatedAt:   round.CreatedAt,
	}
}
