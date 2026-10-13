package service

import (
	"context"

	fulfillmentmodel "ecommerce-be/fulfillment/model"
)

// FulfillmentOrderHooks is the narrow surface fulfillment may call on the
// order module. The order module implements it (order/service/
// order_fulfillment_hooks.go) behind locally-defined signatures that match
// these method-for-method; fulfillment calls it through this interface and
// never touches order repositories. (The compile-time conformance assertion
// lives in the T022 integration test to avoid a production import cycle;
// either direction would forbid future microservice extraction.)
//
// Division of responsibility: the fulfillment applier owns all box rows, so
// IT decides when an order-level transition is due and calls the matching
// hook. Order implementations trust the call, apply the single documented
// transition, and no-op (with an alert) when the order is in an unexpected
// state. Progress carries per-box lines so inventory moves stay exact.
type FulfillmentOrderHooks interface {
	// GetOrderForFulfillment reads the planning inputs for an order:
	// lines, fulfillment type, delivery address snapshot source, COD.
	GetOrderForFulfillment(ctx context.Context, orderID uint) (*fulfillmentmodel.FulfillmentOrderView, error)
	// WithOrderLock runs fn inside a transaction holding a row lock on the
	// order. Planners and draft creators re-check coverage AFTER acquiring
	// it, so concurrent plans cannot both pass the guard.
	WithOrderLock(ctx context.Context, orderID uint, fn func(lockCtx context.Context) error) error
	// OnShipmentsPlanned observes draft creation. No order change.
	OnShipmentsPlanned(ctx context.Context, orderID uint, draftIDs []uint) error
	// OnShipmentBooked observes booking. No order change.
	OnShipmentBooked(ctx context.Context, p fulfillmentmodel.FulfillmentProgress) error
	// OnShipmentDelivered completes the order. Called only when every box
	// is delivered. Order: CONFIRMED → COMPLETED.
	OnShipmentDelivered(ctx context.Context, p fulfillmentmodel.FulfillmentProgress) error
	// OnShipmentFailed ends the order. Order: CONFIRMED → CANCELLED with
	// the reason noted; already-COMPLETED orders are left alone (refund
	// flow owns them) with an alert.
	OnShipmentFailed(ctx context.Context, p fulfillmentmodel.FulfillmentProgress) error
	// OnShipmentReturned returns the order. COMPLETED → RETURNED;
	// CONFIRMED (never completed) → CANCELLED.
	OnShipmentReturned(ctx context.Context, p fulfillmentmodel.FulfillmentProgress) error
}

// FulfillmentInventoryHooks is the narrow surface fulfillment may call on
// the inventory module. Checkout already created the CONFIRMED hold at
// variant level; fulfillment never creates another reservation — it adopts
// (verifies) and releases through these methods.
type FulfillmentInventoryHooks interface {
	// GetAvailability returns available quantity per (variant, location)
	// with warehouse priority and pickup pincode for planning. HeldQty
	// carries this order's own CONFIRMED holds at each spot, so planners
	// allocate from available + held as one pool (holds are earmarked
	// stock the free-availability figure excludes).
	GetAvailability(ctx context.Context, sellerID uint, orderID uint, variantIDs []uint) ([]fulfillmentmodel.AvailabilityRow, error)
	// ReserveForShipment verifies the order's CONFIRMED hold still covers
	// the lines (adopt-only; creates nothing). Shortfall is an error.
	ReserveForShipment(ctx context.Context, sellerID uint, lines []fulfillmentmodel.ReservationLine) error
	// ReleaseReservation returns held units for the lines (cancel / failed
	// pre-pickup paths). Idempotent per (order, variant, location).
	ReleaseReservation(ctx context.Context, sellerID uint, lines []fulfillmentmodel.ReservationLine) error
	// FulfillHolds consumes CONFIRMED holds into FULFILLED on booking
	// (courier custody begins). Partial-qty aware, like release.
	FulfillHolds(ctx context.Context, sellerID uint, orderID uint, lines []fulfillmentmodel.ReservationLine) error
	// UnfulfillHolds returns FULFILLED holds to CONFIRMED when a booked box
	// is cancelled pre-pickup (goods never left). Keeps re-planning working.
	UnfulfillHolds(ctx context.Context, sellerID uint, orderID uint, lines []fulfillmentmodel.ReservationLine) error
}

// FulfillmentProductHooks is the narrow surface fulfillment may call on the
// product module: physical specs normalized to base units (grams, cm).
type FulfillmentProductHooks interface {
	// GetPhysicalSpecs returns normalized specs per variant. Products
	// without specs are simply absent — the caller degrades to manual
	// weight entry (never guessed numbers).
	GetPhysicalSpecs(ctx context.Context, variantIDs []uint) ([]fulfillmentmodel.PhysicalSpec, error)
}
