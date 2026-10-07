package service

import (
	"context"
	"time"

	"ecommerce-be/common/log"
	"ecommerce-be/fulfillment/entity"
	fulfillmentfactory "ecommerce-be/fulfillment/factory"
	fulfillmentmetrics "ecommerce-be/fulfillment/metrics"
	"ecommerce-be/fulfillment/repository"
	courier "ecommerce-be/fulfillment/service/courier"
)

// Staleness thresholds per status family: boxes that outlive them without
// a push or poll get re-polled. Drafts are excluded (recover owns them);
// terminal rows never qualify (the sweep query filters them).
var reconcileStaleAfter = map[entity.ShipmentStatus]time.Duration{
	entity.SHIPMENT_STATUS_BOOKED:           24 * time.Hour,
	entity.SHIPMENT_STATUS_PICKUP_SCHEDULED: 24 * time.Hour,
	entity.SHIPMENT_STATUS_PICKED:           6 * time.Hour,
	entity.SHIPMENT_STATUS_IN_TRANSIT:       6 * time.Hour,
	entity.SHIPMENT_STATUS_OUT_FOR_DELIVERY: 6 * time.Hour,
	entity.SHIPMENT_STATUS_NDR_PENDING:      6 * time.Hour,
	entity.SHIPMENT_STATUS_RTO_IN_TRANSIT:   6 * time.Hour,
}

// reconcileBatchSize caps one sweep tick (bulk API batches per account).
const reconcileBatchSize = 100

// ReconcileService heals missed webhooks: stale in-flight boxes are
// re-polled in bulk per account and applied through the shared path. It
// also sweeps stuck NDR rounds (open > 24h without seller action).
type ReconcileService interface {
	ReconcilePending(ctx context.Context) (int, error)
	SweepNDR(ctx context.Context) (int, error)
}

// ReconcileServiceImpl implements ReconcileService.
type ReconcileServiceImpl struct {
	shipmentRepo repository.ShipmentRepository
	ndrRepo      repository.NDRRepository
	applier      *NormalizedApplier
	factory      *fulfillmentfactory.CourierPartnerFactory
	// metrics is the optional sweep summary recorder (T059). Nil disables
	// recording; wired by the factory via SetMetricsRecorder.
	metrics fulfillmentmetrics.Recorder
}

// SetMetricsRecorder attaches the sweep summary recorder. Safe to call
// with nil (disables recording).
func (s *ReconcileServiceImpl) SetMetricsRecorder(r fulfillmentmetrics.Recorder) {
	s.metrics = r
}

// recordSweep emits one sweep summary metric; nil-safe.
func (s *ReconcileServiceImpl) recordSweep(
	ctx context.Context,
	sweep, result string,
	count int,
	start time.Time,
) {
	if s.metrics == nil {
		return
	}
	s.metrics.RecordSweep(ctx, sweep, "", result, count, time.Since(start))
}

// NewReconcileService builds the reconciler. The applier is shared with the
// webhook pipeline, so cron and push can never diverge.
func NewReconcileService(
	shipmentRepo repository.ShipmentRepository,
	ndrRepo repository.NDRRepository,
	applier *NormalizedApplier,
	factory *fulfillmentfactory.CourierPartnerFactory,
) ReconcileService {
	return &ReconcileServiceImpl{
		shipmentRepo: shipmentRepo,
		ndrRepo:      ndrRepo,
		applier:      applier,
		factory:      factory,
	}
}

// ReconcilePending sweeps stale boxes grouped by frozen config (one bulk
// call per account) and applies results as source=system. Returns healed
// (moved) count.
func (s *ReconcileServiceImpl) ReconcilePending(ctx context.Context) (int, error) {
	start := time.Now()
	statuses := make([]entity.ShipmentStatus, 0, len(reconcileStaleAfter))
	oldest := time.Now().UTC()
	for status, threshold := range reconcileStaleAfter {
		statuses = append(statuses, status)
		if cutoff := time.Now().UTC().Add(-threshold); cutoff.Before(oldest) {
			oldest = cutoff
		}
	}
	// One query with the loosest cutoff; per-status thresholds filter below.
	stale, err := s.shipmentRepo.FindStaleInFlight(ctx, statuses, oldest, reconcileBatchSize)
	if err != nil {
		s.recordSweep(ctx, "reconcile_pending", fulfillmentmetrics.SweepError, 0, start)
		return 0, err
	}

	byConfig := map[uint][]*entity.FulfillmentShipment{}
	for i := range stale {
		shipment := &stale[i]
		if !s.isStale(shipment) {
			continue
		}
		if shipment.ProviderConfigID == nil || shipment.AWB == nil {
			continue
		}
		byConfig[*shipment.ProviderConfigID] = append(byConfig[*shipment.ProviderConfigID], shipment)
	}

	healed := 0
	for configID, group := range byConfig {
		healed += s.reconcileAccount(ctx, configID, group)
	}
	s.recordSweep(ctx, "reconcile_pending", fulfillmentmetrics.SweepOK, healed, start)
	return healed, nil
}

// isStale applies the per-status threshold (the query used the loosest).
func (s *ReconcileServiceImpl) isStale(shipment *entity.FulfillmentShipment) bool {
	threshold, ok := reconcileStaleAfter[shipment.Status]
	if !ok {
		return false
	}
	if shipment.LastSyncedAt == nil {
		return true
	}
	return time.Since(*shipment.LastSyncedAt) > threshold
}

// reconcileAccount bulk-polls one account and applies each result. One bad
// box never blocks its siblings; errors are logged per box.
func (s *ReconcileServiceImpl) reconcileAccount(
	ctx context.Context,
	configID uint,
	group []*entity.FulfillmentShipment,
) int {
	resolved, err := s.factory.ResolveConfig(ctx, configID)
	if err != nil {
		log.ErrorWithContext(ctx, "reconcile: account unresolvable, skipping group", err)
		return 0
	}
	awbs := make([]string, 0, len(group))
	byAWB := map[string]*entity.FulfillmentShipment{}
	for _, shipment := range group {
		awb := ""
		if shipment.AWB != nil {
			awb = *shipment.AWB
		}
		if awb == "" {
			continue
		}
		awbs = append(awbs, awb)
		byAWB[awb] = shipment
	}
	if len(awbs) == 0 {
		return 0
	}

	events, err := s.fetchBulk(ctx, resolved, awbs)
	if err != nil {
		log.ErrorWithContext(ctx, "reconcile: bulk poll failed, next tick retries", err)
		return 0
	}
	healed := 0
	for _, event := range events {
		shipment, ok := byAWB[event.AWB]
		if !ok || event == nil {
			continue
		}
		result, err := s.applier.Apply(ctx, shipment, event, EventSourceSystem)
		if err != nil {
			log.ErrorWithContext(ctx, "reconcile: apply failed, next tick retries", err)
			continue
		}
		if result.Moved {
			healed++
		}
	}
	return healed
}

// fetchBulk prefers the bulk endpoint, falling back to single polls when
// the adapter lacks bulk (direct couriers without one).
func (s *ReconcileServiceImpl) fetchBulk(
	ctx context.Context,
	resolved *fulfillmentfactory.ResolvedCredentials,
	awbs []string,
) ([]*courier.NormalizedShipmentEvent, error) {
	if bulk, ok := resolved.Adapter.(courierBulkTracker); ok {
		return bulk.FetchTrackingBulk(ctx, awbs, resolved.Creds)
	}
	events := make([]*courier.NormalizedShipmentEvent, 0, len(awbs))
	for _, awb := range awbs {
		event, err := resolved.Adapter.FetchTracking(ctx, awb, resolved.Creds)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

// courierBulkTracker mirrors the optional bulk capability structurally so
// this file never imports the courier contract package.
type courierBulkTracker interface {
	FetchTrackingBulk(ctx context.Context, awbs []string, creds map[string]any) ([]*courier.NormalizedShipmentEvent, error)
}

// ndrEscalationAge is how long an open round waits for seller action before
// the sweep escalates (dashboard-visible alert via error log; push channels
// are deferred per the v1 scope decision).
const ndrEscalationAge = 24 * time.Hour

// SweepNDR re-pulls open NDR rounds through capable adapters and escalates
// rounds still unactioned past the age threshold. Escalation is a loud log
// plus the dashboard-visible open round itself (push channels deferred).
// Returns escalated count.
func (s *ReconcileServiceImpl) SweepNDR(ctx context.Context) (int, error) {
	start := time.Now()
	rounds, err := s.ndrRepo.FindStaleOpen(ctx, time.Now().UTC().Add(-ndrEscalationAge), reconcileBatchSize)
	if err != nil {
		s.recordSweep(ctx, "ndr_sweep", fulfillmentmetrics.SweepError, 0, start)
		return 0, err
	}
	escalated := 0
	for i := range rounds {
		round := &rounds[i]
		shipment, err := s.shipmentRepo.FindByID(ctx, round.ShipmentID)
		if err != nil {
			log.WithContext(ctx).
				WithField("shipmentId", round.ShipmentID).
				WithField("error", err.Error()).
				Error("ndr sweep: shipment unreadable")
			continue
		}
		if shipment.ProviderConfigID == nil {
			continue
		}
		resolved, err := s.factory.ResolveConfig(ctx, *shipment.ProviderConfigID)
		if err != nil {
			log.WithContext(ctx).
				WithField("shipmentId", round.ShipmentID).
				WithField("error", err.Error()).
				Error("ndr sweep: account unresolvable")
			continue
		}
		if handler, ok := resolved.Adapter.(courierNDRReader); ok && shipment.AWB != nil {
			if _, err := handler.GetNDR(ctx, *shipment.AWB, resolved.Creds); err != nil {
				log.WithContext(ctx).
					WithField("shipmentId", round.ShipmentID).
					WithField("awb", *shipment.AWB).
					WithField("error", err.Error()).
					Error("ndr sweep: provider re-pull failed")
			}
		}
		log.WithContext(ctx).
			WithField("shipmentId", round.ShipmentID).
			WithField("awb", round.AWB).
			WithField("attempt", round.AttemptNo).
			Error("ndr sweep: round unactioned past escalation age")
		escalated++
	}
	s.recordSweep(ctx, "ndr_sweep", fulfillmentmetrics.SweepOK, escalated, start)
	return escalated, nil
}

// courierNDRReader mirrors the NDR read capability structurally.
type courierNDRReader interface {
	GetNDR(ctx context.Context, awb string, creds map[string]any) (any, error)
}
