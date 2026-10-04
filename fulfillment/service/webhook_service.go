package service

import (
	"context"
	"net/http"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/fulfillment/entity"
	fulfillmenterrors "ecommerce-be/fulfillment/error"
	fulfillmentfactory "ecommerce-be/fulfillment/factory"
	fulfillmentmetrics "ecommerce-be/fulfillment/metrics"
	"ecommerce-be/fulfillment/repository"
	courier "ecommerce-be/fulfillment/service/courier"
)

// WebhookService processes verified inbound courier pushes.
type WebhookService interface {
	// HandleWebhook verifies, dedupes, and applies one push. It returns a
	// sentinel error only for 404 (unknown provider) and 401 (bad
	// signature); every verified outcome — applied, duplicate, ignored,
	// even apply-failed — returns nil (HTTP 200) so the courier stops
	// retrying. Apply failures persist as failed logs for the reconciler.
	HandleWebhook(
		ctx context.Context,
		providerCode string,
		rawBody []byte,
		headers http.Header,
		ipAddress string,
	) error
}

// WebhookServiceImpl implements WebhookService.
type WebhookServiceImpl struct {
	providerRepo repository.CourierProviderRepository
	shipmentRepo repository.ShipmentRepository
	webhookRepo  repository.WebhookLogRepository
	factory      *fulfillmentfactory.CourierPartnerFactory
	applier      *NormalizedApplier
	// metrics is the optional webhook outcome recorder (T059). Nil disables
	// recording; wired by the factory via SetMetricsRecorder.
	metrics fulfillmentmetrics.Recorder
}

// SetMetricsRecorder attaches the webhook outcome recorder. Safe to call
// with nil (disables recording).
func (s *WebhookServiceImpl) SetMetricsRecorder(r fulfillmentmetrics.Recorder) {
	s.metrics = r
}

// recordOutcome emits one webhook metric; nil-safe.
func (s *WebhookServiceImpl) recordOutcome(
	ctx context.Context,
	providerCode, outcome, action string,
	start time.Time,
) {
	if s.metrics == nil {
		return
	}
	s.metrics.RecordWebhook(ctx, providerCode, outcome, action, time.Since(start))
}

// NewWebhookService creates the webhook service.
func NewWebhookService(
	providerRepo repository.CourierProviderRepository,
	shipmentRepo repository.ShipmentRepository,
	webhookRepo repository.WebhookLogRepository,
	factory *fulfillmentfactory.CourierPartnerFactory,
	applier *NormalizedApplier,
) WebhookService {
	return &WebhookServiceImpl{
		providerRepo: providerRepo,
		shipmentRepo: shipmentRepo,
		webhookRepo:  webhookRepo,
		factory:      factory,
		applier:      applier,
	}
}

// HandleWebhook verifies, dedupes, and applies one courier push.
//
// Locate-then-verify: untrusted locator hints select the shipment first;
// the signature is then checked with THAT shipment's frozen account
// (never a first-config guess). Unverifiable or unlocatable pushes change
// no state and persist nothing.
func (s *WebhookServiceImpl) HandleWebhook(
	ctx context.Context,
	providerCode string,
	rawBody []byte,
	headers http.Header,
	ipAddress string,
) error {
	start := time.Now()
	// 1. Resolve the provider record + adapter. Unknown code → 404.
	if _, err := s.providerRepo.FindByCode(ctx, providerCode); err != nil {
		s.recordOutcome(ctx, providerCode, fulfillmentmetrics.WebhookError, "", start)
		return err
	}
	adapter, err := s.factory.GetByCode(providerCode)
	if err != nil {
		s.recordOutcome(ctx, providerCode, fulfillmentmetrics.WebhookError, "", start)
		return err
	}

	// 2. Peek untrusted locators and find our shipment. No match → 200
	// with NO persistence (booking may still be in flight).
	shipment, err := s.locateShipment(ctx, providerCode, adapter.PeekLocators(rawBody))
	if err != nil {
		s.recordOutcome(ctx, providerCode, fulfillmentmetrics.WebhookError, "", start)
		return err
	}
	if shipment == nil {
		s.recordOutcome(ctx, providerCode, fulfillmentmetrics.WebhookUnknownAWB, "", start)
		return nil
	}

	// 3. Load the frozen config row the shipment booked with.
	if shipment.ProviderConfigID == nil {
		s.recordOutcome(ctx, providerCode, fulfillmentmetrics.WebhookError, "", start)
		return fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"shipment without provider account")
	}
	resolved, err := s.factory.ResolveConfig(ctx, *shipment.ProviderConfigID)
	if err != nil {
		s.recordOutcome(ctx, providerCode, fulfillmentmetrics.WebhookError, "", start)
		return err
	}

	// 4. Verify (constant-time secret check on raw bytes) and normalize.
	// Bad signature → 401, nothing stored.
	normalized, err := adapter.NormalizeWebhook(rawBody, headers, resolved.Creds)
	if err != nil {
		s.recordOutcome(ctx, providerCode, fulfillmentmetrics.WebhookUnverified, "", start)
		return fulfillmenterrors.ErrorWebhookUnverified.WithMessagef("%s", firstLine(err.Error()))
	}
	action := string(normalized.Action)

	// 5. Record the log; the unique (provider_code, event_id) index turns
	// replays into a 200 no-op. The shipment link is set up front so every
	// outcome stays attributable to the seller's ledger.
	logEntry := &entity.FulfillmentWebhookLog{
		ProviderCode: providerCode,
		EventID:      normalized.EventID,
		AWB:          strOrNil(normalized.AWB),
		Action:       string(normalized.Action),
		Status:       "received",
		Payload:      db.JSONMap(normalized.Payload),
		ShipmentID:   &shipment.ID,
		IPAddress:    strOrNil(ipAddress),
	}

	var applyErr error
	outcome := fulfillmentmetrics.WebhookApplied
	if err := db.WithTransaction(ctx, func(txCtx context.Context) error {
		duplicate, err := s.webhookRepo.Create(txCtx, logEntry)
		if err != nil {
			return err
		}
		if duplicate {
			outcome = fulfillmentmetrics.WebhookDuplicate
			return nil
		}
		if normalized.Action == courier.ShipmentActionIgnore {
			outcome = fulfillmentmetrics.WebhookIgnored
			return s.webhookRepo.MarkIgnored(txCtx, logEntry.ID)
		}
		result, err := s.applier.Apply(txCtx, shipment, normalized, EventSourceWebhook)
		if err != nil {
			// Persist the failure and SWALLOW it inside the transaction:
			// returning err here would roll back the failed-status row.
			// The cron heals from that row.
			if markErr := s.webhookRepo.MarkFailed(txCtx, logEntry.ID, err.Error()); markErr != nil {
				return markErr
			}
			outcome = fulfillmentmetrics.WebhookApplyFailed
			applyErr = err
			return nil
		}
		if result.Moved {
			return s.webhookRepo.MarkProcessed(txCtx, logEntry.ID, shipment.ID)
		}
		outcome = fulfillmentmetrics.WebhookIgnored
		return s.webhookRepo.MarkIgnored(txCtx, logEntry.ID)
	}); err != nil {
		s.recordOutcome(ctx, providerCode, fulfillmentmetrics.WebhookError, action, start)
		return err
	}
	s.recordOutcome(ctx, providerCode, outcome, action, start)
	_ = applyErr
	return nil
}

// locateShipment finds the shipment matching untrusted webhook hints
// within one provider's namespace. Priority: AWB (per-box unique) →
// provider order id. Absent rows are normal (booking in flight, unknown
// ids) and yield (nil, nil) — never an error, so the caller answers 200
// with no persistence. Genuine DB failures propagate.
func (s *WebhookServiceImpl) locateShipment(
	ctx context.Context,
	providerCode string,
	loc courier.TrackLocators,
) (*entity.FulfillmentShipment, error) {
	if loc.AWB != "" {
		shipment, err := s.shipmentRepo.FindByAWB(ctx, providerCode, loc.AWB)
		if err != nil {
			if err == fulfillmenterrors.ErrorFulfillmentNotFound {
				return nil, nil
			}
			return nil, err
		}
		return shipment, nil
	}
	if loc.ProviderOrderID != "" {
		shipment, err := s.shipmentRepo.FindByProviderOrderID(ctx, providerCode, loc.ProviderOrderID)
		if err != nil {
			if err == fulfillmenterrors.ErrorFulfillmentNotFound {
				return nil, nil
			}
			return nil, err
		}
		return shipment, nil
	}
	return nil, nil
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
