package courier

import (
	"context"
	"net/http"
	"time"

	"ecommerce-be/fulfillment/model"
)

// CourierPartner is the single provider contract every courier adapter
// implements. Base services (shipment, rate, tracking, webhook, reconcile,
// NDR, return) speak ONLY to this interface, resolved by provider code
// through CourierPartnerFactory. Adding a new courier is purely additive:
// a new folder implementing this interface + seed rows + one factory
// registry line. Orchestrators must never switch on provider codes or
// import provider subpackages.
type CourierPartner interface {
	Code() string

	// Credentials (seller dashboard + decrypt for outbound calls).
	Validate(raw map[string]any, partial bool) error
	Encrypt(raw map[string]any) (map[string]any, error)
	Decrypt(stored map[string]any) (map[string]any, error)
	MaskHints(stored map[string]any) map[string]any
	MergePartial(existingStored, incoming map[string]any) (map[string]any, error)

	// GetRates shops live rates. Read-only: safe to retry and cache.
	GetRates(ctx context.Context, in model.RateInput, creds map[string]any) ([]model.RateOption, error)

	// BookShipment sends our shipment id as the provider order id and
	// assigns the AWB. If in.PickupAt is set, it schedules pickup too.
	// POST-equivalent: NEVER retried by the HTTP layer (provider dedupes
	// on our shipment id; a retried POST risks double AWBs).
	BookShipment(ctx context.Context, in model.BookShipmentInput, creds map[string]any) (*model.BookShipmentOutput, error)
	Cancel(ctx context.Context, in model.CancelInput, creds map[string]any) error
	GetLabel(ctx context.Context, in model.LabelInput, creds map[string]any) ([]byte, error)

	// FetchTracking polls one AWB; FetchTrackingBulk is the cron path
	// (batched per account). Both are safe to retry.
	FetchTracking(ctx context.Context, awb string, creds map[string]any) (*NormalizedShipmentEvent, error)
	FetchTrackingBulk(ctx context.Context, awbs []string, creds map[string]any) ([]*NormalizedShipmentEvent, error)

	// PeekLocators scrapes locator hints from a raw webhook body WITHOUT
	// verifying authenticity. Hints are used only to find which shipment
	// (and therefore which seller secret) to verify with; they are discarded
	// if signature verification fails.
	PeekLocators(rawBody []byte) TrackLocators

	// NormalizeWebhook verifies the request signature from headers + creds,
	// then maps the provider event to a frozen ShipmentAction.
	NormalizeWebhook(rawBody []byte, headers http.Header, creds map[string]any) (*NormalizedShipmentEvent, error)

	// TestConnection verifies credentials with a lightweight authenticated
	// call. Provider 401 surfaces as a validation error (400-class), not a
	// server failure. Nothing is persisted.
	TestConnection(ctx context.Context, creds map[string]any) error
}

// PickupScheduler is an OPTIONAL adapter capability for couriers where
// pickup scheduling is a separate call from booking. The service
// type-asserts and returns CAPABILITY_UNSUPPORTED when absent.
type PickupScheduler interface {
	SchedulePickup(ctx context.Context, in model.PickupInput, creds map[string]any) error
}

// NDRHandler is an OPTIONAL adapter capability for failed-delivery rounds.
// Direct couriers often lack NDR; aggregators provide it.
type NDRHandler interface {
	GetNDR(ctx context.Context, awb string, creds map[string]any) (*model.NDRDetail, error)
	ActNDR(ctx context.Context, in model.NDRActionInput, creds map[string]any) error
}

// ReturnHandler is an OPTIONAL adapter capability for return/RTO booking.
type ReturnHandler interface {
	BookReturn(ctx context.Context, in model.BookReturnInput, creds map[string]any) (*model.BookShipmentOutput, error)
}

// RTORequester is an OPTIONAL adapter capability for proactive mid-transit
// return requests (no open NDR round). Distinct from NDR-action RTO.
type RTORequester interface {
	RequestRTO(ctx context.Context, in model.RTORequestInput, creds map[string]any) error
}

// TrackLocators are the untrusted shipment hints scraped from a raw webhook body.
type TrackLocators struct {
	AWB             string
	ProviderOrderID string
}

// ShipmentAction is the frozen set of normalized outcomes base services may
// switch on. New couriers map their own event names to these actions inside
// their adapter folder — the base switch never grows.
type ShipmentAction string

const (
	ShipmentActionIgnore          ShipmentAction = "ignore"
	ShipmentActionBooked          ShipmentAction = "booked"
	ShipmentActionPickupScheduled ShipmentAction = "pickup_scheduled"
	ShipmentActionPicked          ShipmentAction = "picked"
	ShipmentActionInTransit       ShipmentAction = "in_transit"
	ShipmentActionOutForDelivery  ShipmentAction = "out_for_delivery"
	ShipmentActionNDRPending      ShipmentAction = "ndr_pending"
	ShipmentActionDelivered       ShipmentAction = "delivered"
	ShipmentActionFailed          ShipmentAction = "failed"
	ShipmentActionRTOInTransit    ShipmentAction = "rto_in_transit"
	ShipmentActionReturned        ShipmentAction = "returned"
	ShipmentActionCancelled       ShipmentAction = "cancelled"
)

// NormalizedShipmentEvent is the provider-agnostic result of verifying
// (webhook) or polling (cron/refresh) a provider event. Apply paths switch
// ONLY on Action.
type NormalizedShipmentEvent struct {
	Action                              ShipmentAction
	EventID                             string // idempotency key (provider unique; never empty)
	ProviderEvent                       string // raw provider label, stored on the ledger for audit only
	AWB                                 string
	ProviderOrderID, ProviderShipmentID string
	CourierName                         string
	ETD                                 *time.Time
	FailureCode                         string
	FailureMessage                      string
	Payload                             map[string]any // sanitized for storage (no phones/addresses)
}
