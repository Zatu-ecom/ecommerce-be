package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ecommerce-be/common/log"
	"ecommerce-be/fulfillment/entity"
	fulfillmenterrors "ecommerce-be/fulfillment/error"
	"ecommerce-be/fulfillment/model"
	"ecommerce-be/fulfillment/repository"
	courier "ecommerce-be/fulfillment/service/courier"
)

// Event sources recorded on the ledger (match the source column vocabulary).
const (
	EventSourceWebhook = "webhook"
	EventSourceSystem  = "system"
)

// allowedTransitions is the data-model §2a allow-list: movement is never a
// ladder (out_for_delivery ↔ in_transit, NDR loops), and terminal rows have
// no outgoing edges. Verified but disallowed pairs are logged as ignore.
var allowedTransitions = map[entity.ShipmentStatus][]entity.ShipmentStatus{
	entity.SHIPMENT_STATUS_DRAFT: {
		entity.SHIPMENT_STATUS_BOOKED,
		entity.SHIPMENT_STATUS_CANCELLED,
	},
	entity.SHIPMENT_STATUS_BOOKED: {
		// Terminal states are reachable directly: missed pushes mean the
		// first observed event can be delivery/failure, and the reconciler
		// exists precisely for those jumps. RTO stays out: returning
		// before pickup is a cancel, not an RTO leg.
		entity.SHIPMENT_STATUS_PICKUP_SCHEDULED,
		entity.SHIPMENT_STATUS_PICKED,
		entity.SHIPMENT_STATUS_IN_TRANSIT,
		entity.SHIPMENT_STATUS_OUT_FOR_DELIVERY,
		entity.SHIPMENT_STATUS_NDR_PENDING,
		entity.SHIPMENT_STATUS_DELIVERED,
		entity.SHIPMENT_STATUS_CANCELLED,
		entity.SHIPMENT_STATUS_FAILED,
	},
	entity.SHIPMENT_STATUS_PICKUP_SCHEDULED: {
		entity.SHIPMENT_STATUS_PICKED,
		entity.SHIPMENT_STATUS_IN_TRANSIT,
		entity.SHIPMENT_STATUS_OUT_FOR_DELIVERY,
		entity.SHIPMENT_STATUS_NDR_PENDING,
		entity.SHIPMENT_STATUS_DELIVERED,
		entity.SHIPMENT_STATUS_CANCELLED,
		entity.SHIPMENT_STATUS_FAILED,
	},
	entity.SHIPMENT_STATUS_PICKED: {
		entity.SHIPMENT_STATUS_IN_TRANSIT,
		entity.SHIPMENT_STATUS_OUT_FOR_DELIVERY,
		entity.SHIPMENT_STATUS_NDR_PENDING,
		entity.SHIPMENT_STATUS_RTO_IN_TRANSIT,
		entity.SHIPMENT_STATUS_DELIVERED,
		entity.SHIPMENT_STATUS_FAILED,
	},
	entity.SHIPMENT_STATUS_IN_TRANSIT: {
		entity.SHIPMENT_STATUS_OUT_FOR_DELIVERY,
		entity.SHIPMENT_STATUS_NDR_PENDING,
		entity.SHIPMENT_STATUS_RTO_IN_TRANSIT,
		entity.SHIPMENT_STATUS_DELIVERED,
		entity.SHIPMENT_STATUS_FAILED,
	},
	entity.SHIPMENT_STATUS_OUT_FOR_DELIVERY: {
		entity.SHIPMENT_STATUS_IN_TRANSIT,
		entity.SHIPMENT_STATUS_NDR_PENDING,
		entity.SHIPMENT_STATUS_RTO_IN_TRANSIT,
		entity.SHIPMENT_STATUS_DELIVERED,
		entity.SHIPMENT_STATUS_FAILED,
	},
	entity.SHIPMENT_STATUS_NDR_PENDING: {
		entity.SHIPMENT_STATUS_IN_TRANSIT,
		entity.SHIPMENT_STATUS_OUT_FOR_DELIVERY,
		entity.SHIPMENT_STATUS_RTO_IN_TRANSIT,
		entity.SHIPMENT_STATUS_FAILED,
	},
	entity.SHIPMENT_STATUS_RTO_IN_TRANSIT: {
		entity.SHIPMENT_STATUS_RETURNED,
		entity.SHIPMENT_STATUS_FAILED,
	},
}

// ApplyResult reports what one apply did for the caller's bookkeeping
// (webhook-log linkage, cron metrics).
type ApplyResult struct {
	ShipmentID uint
	Moved      bool
}

// NormalizedApplier applies verified normalized events to shipments. It is
// the ONE apply path shared by the webhook pipeline (source=webhook) and
// the reconciliation cron (source=system), so the two can never diverge on
// the state machine, idempotency, or order hooks.
//
// The switch is ONLY on courier.ShipmentAction — provider event names never
// reach this file.
type NormalizedApplier struct {
	shipments repository.ShipmentRepository
	items     repository.ShipmentItemRepository
	events    repository.ShipmentEventRepository
	ndr       repository.NDRRepository
	orders    FulfillmentOrderHooks
}

// NewNormalizedApplier creates the shared apply path.
func NewNormalizedApplier(
	shipments repository.ShipmentRepository,
	items repository.ShipmentItemRepository,
	events repository.ShipmentEventRepository,
	ndr repository.NDRRepository,
	orders FulfillmentOrderHooks,
) *NormalizedApplier {
	return &NormalizedApplier{
		shipments: shipments,
		items:     items,
		events:    events,
		ndr:       ndr,
		orders:    orders,
	}
}

// Apply applies one normalized event to an already-located shipment.
func (a *NormalizedApplier) Apply(
	ctx context.Context,
	shipment *entity.FulfillmentShipment,
	event *courier.NormalizedShipmentEvent,
	source string,
) (*ApplyResult, error) {
	result := &ApplyResult{ShipmentID: shipment.ID}

	// AWB mismatch: the event does not describe this box (wrong-box push).
	if event.AWB != "" && shipment.AWB != nil && *shipment.AWB != "" &&
		event.AWB != *shipment.AWB {
		return result, fulfillmenterrors.ErrorApplyMismatch
	}

	// Terminal rows never move; repeats of the current status write no
	// ledger row (idempotent no-op, no event spam).
	if shipment.Status.IsTerminal() {
		return result, nil
	}
	if event.Action == courier.ShipmentActionIgnore {
		if err := a.observe(ctx, shipment, event, source); err != nil {
			return result, err
		}
		return result, nil
	}
	to := entity.ShipmentStatus(event.Action)
	// NDR scans route through round management even when the status does
	// not change: a new scan after the previous round closed opens the next
	// attempt; scans of an open round are idempotent no-ops.
	if to == entity.SHIPMENT_STATUS_NDR_PENDING {
		return a.applyNDR(ctx, shipment, event, source, result)
	}
	if to == shipment.Status {
		return result, nil
	}
	if !allowed(shipment.Status, to) {
		if err := a.observe(ctx, shipment, event, source); err != nil {
			return result, err
		}
		return result, nil
	}

	if err := a.move(ctx, shipment, event, to, source); err != nil {
		return result, err
	}
	result.Moved = true

	return result, a.afterMove(ctx, shipment, event, to)
}

// allowed reports whether the transition is in the allow-list.
func allowed(from, to entity.ShipmentStatus) bool {
	for _, next := range allowedTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// observe records a ledger observation without moving status (ignore
// actions and disallowed pairs).
func (a *NormalizedApplier) observe(
	ctx context.Context,
	shipment *entity.FulfillmentShipment,
	event *courier.NormalizedShipmentEvent,
	source string,
) error {
	if err := a.events.Create(ctx, &entity.FulfillmentShipmentEvent{
		ShipmentID:     shipment.ID,
		EventType:      string(courier.ShipmentActionIgnore),
		FromStatus:     strPtr(string(shipment.Status)),
		ToStatus:       string(shipment.Status),
		ProviderEvent:  &event.ProviderEvent,
		FailureCode:    nonEmptyPtr(event.FailureCode),
		FailureMessage: nonEmptyPtr(event.FailureMessage),
		Source:         source,
	}); err != nil {
		return fmt.Errorf("apply observe ledger: %w", err)
	}
	return nil
}

// move persists the status transition with timestamp and courier patches.
func (a *NormalizedApplier) move(
	ctx context.Context,
	shipment *entity.FulfillmentShipment,
	event *courier.NormalizedShipmentEvent,
	to entity.ShipmentStatus,
	source string,
) error {
	now := time.Now().UTC()
	patch := map[string]any{"last_synced_at": now}
	if to == entity.SHIPMENT_STATUS_PICKED && shipment.ShippedAt == nil {
		patch["shipped_at"] = now
	}
	if to == entity.SHIPMENT_STATUS_DELIVERED {
		patch["delivered_at"] = now
	}
	if to == entity.SHIPMENT_STATUS_CANCELLED {
		patch["cancelled_at"] = now
	}
	if event.CourierName != "" {
		patch["courier_name"] = event.CourierName
	}

	moved, err := a.shipments.UpdateStatusIfCurrent(ctx, shipment.ID, shipment.Status, to, patch)
	if err != nil {
		return fmt.Errorf("apply move: %w", err)
	}
	if !moved {
		// Lost a race with a parallel apply; the winner's state stands.
		return nil
	}
	from := string(shipment.Status)
	shipment.Status = to
	return a.events.Create(ctx, &entity.FulfillmentShipmentEvent{
		ShipmentID:     shipment.ID,
		EventType:      string(to),
		FromStatus:     &from,
		ToStatus:       string(to),
		ProviderEvent:  &event.ProviderEvent,
		FailureCode:    nonEmptyPtr(event.FailureCode),
		FailureMessage: nonEmptyPtr(event.FailureMessage),
		Source:         source,
	})
}

// afterMove runs post-transition side effects: order hooks with exact box
// lines. (NDR rounds are handled by applyNDR, which runs instead of the
// generic move path.)
func (a *NormalizedApplier) afterMove(
	ctx context.Context,
	shipment *entity.FulfillmentShipment,
	event *courier.NormalizedShipmentEvent,
	to entity.ShipmentStatus,
) error {
	var err error
	switch to {
	case entity.SHIPMENT_STATUS_DELIVERED:
		// Return boxes completing their journey restock, not complete:
		// delivered on a return box means back at the warehouse.
		if shipment.ReturnOfShipmentID != nil {
			err = a.orders.OnShipmentReturned(ctx, a.progress(ctx, shipment, event))
		} else {
			err = a.orders.OnShipmentDelivered(ctx, a.progress(ctx, shipment, event))
		}
	case entity.SHIPMENT_STATUS_FAILED, entity.SHIPMENT_STATUS_CANCELLED:
		err = a.orders.OnShipmentFailed(ctx, a.progress(ctx, shipment, event))
	case entity.SHIPMENT_STATUS_RETURNED:
		err = a.orders.OnShipmentReturned(ctx, a.progress(ctx, shipment, event))
	}
	if err != nil {
		return fmt.Errorf("apply order hook: %w", err)
	}
	return nil
}

// applyNDR routes failed-delivery scans: moves status when legal, then
// ensures exactly one open round (new scans of an open round are no-ops;
// scans after action open the next attempt).
func (a *NormalizedApplier) applyNDR(
	ctx context.Context,
	shipment *entity.FulfillmentShipment,
	event *courier.NormalizedShipmentEvent,
	source string,
	result *ApplyResult,
) (*ApplyResult, error) {
	if shipment.Status != entity.SHIPMENT_STATUS_NDR_PENDING {
		if !allowed(shipment.Status, entity.SHIPMENT_STATUS_NDR_PENDING) {
			return a.observeAndReturn(ctx, shipment, event, source, result)
		}
		if err := a.move(ctx, shipment, event, entity.SHIPMENT_STATUS_NDR_PENDING, source); err != nil {
			return result, err
		}
		result.Moved = true
	}
	if err := a.openNDRRound(ctx, shipment, event); err != nil {
		return result, err
	}
	return result, nil
}

func (a *NormalizedApplier) observeAndReturn(
	ctx context.Context,
	shipment *entity.FulfillmentShipment,
	event *courier.NormalizedShipmentEvent,
	source string,
	result *ApplyResult,
) (*ApplyResult, error) {
	if err := a.observe(ctx, shipment, event, source); err != nil {
		return result, err
	}
	return result, nil
}

// openNDRRound starts the next attempt round, unless one is already open
// (redelivered scans of the same round are idempotent no-ops).
func (a *NormalizedApplier) openNDRRound(
	ctx context.Context,
	shipment *entity.FulfillmentShipment,
	event *courier.NormalizedShipmentEvent,
) error {
	open, err := a.ndr.FindOpenByShipment(ctx, shipment.ID)
	if err != nil {
		return err
	}
	if open != nil {
		return nil
	}
	maxAttempt, err := a.ndr.MaxAttemptNo(ctx, shipment.ID)
	if err != nil {
		return err
	}
	reason := event.ProviderEvent
	if event.FailureMessage != "" {
		reason = event.FailureMessage
	}
	if err := a.ndr.Create(ctx, &entity.FulfillmentNDR{
		ShipmentID: shipment.ID,
		AWB:        event.AWB,
		AttemptNo:  maxAttempt + 1,
		NDRStatus:  event.ProviderEvent,
		Reason:     &reason,
	}); err != nil {
		// Concurrent double-open: the partial unique index rejects the
		// loser; that is the correct outcome, not an error.
		if isUniqueViolation(err) {
			log.InfoWithContext(ctx, "apply: concurrent NDR open deduplicated")
			return nil
		}
		return fmt.Errorf("apply open NDR round: %w", err)
	}
	return nil
}

// progress builds the per-box hook payload from live box lines.
func (a *NormalizedApplier) progress(
	ctx context.Context,
	shipment *entity.FulfillmentShipment,
	event *courier.NormalizedShipmentEvent,
) model.FulfillmentProgress {
	progress := model.FulfillmentProgress{
		OrderID:          shipment.OrderID,
		ShipmentID:       shipment.ID,
		Reason:           event.FailureMessage,
		PickupLocationID: derefUint(shipment.PickupLocationID),
	}
	items, err := a.items.FindByShipmentID(ctx, shipment.ID)
	if err != nil {
		log.ErrorWithContext(ctx, "apply: lines unreadable, hooks run lineless", err)
		return progress
	}
	for _, item := range items {
		progress.Items = append(progress.Items, model.FulfillmentLine{
			OrderItemID: item.OrderItemID,
			Quantity:    item.Quantity,
		})
	}
	return progress
}

func nonEmptyPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// isUniqueViolation reports Postgres unique violations (23505), used for
// benign concurrent-insert races the schema already arbitrates.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "23505")
}
