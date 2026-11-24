package service

import (
	"context"
	"fmt"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/fulfillment/entity"
	fulfillmenterrors "ecommerce-be/fulfillment/error"
	fulfillmentfactory "ecommerce-be/fulfillment/factory"
	"ecommerce-be/fulfillment/model"
	"ecommerce-be/fulfillment/repository"
)

// TrackingService serves reads: PG-only customer tracking, seller refresh
// (fetch + apply), and the seller's webhook audit log.
type TrackingService interface {
	// GetCustomerTrack returns a buyer's boxes for one order: PG status +
	// ledger, sanitized for customers. Never calls the courier, never
	// changes state.
	GetCustomerTrack(ctx context.Context, userID uint, orderID uint) ([]model.ShipmentResponse, error)
	// RefreshTrack pulls live provider state for one box and applies it
	// through the shared path. Singleflight per shipment.
	RefreshTrack(ctx context.Context, sellerID uint, shipmentID uint) (model.ShipmentResponse, error)
	// ListWebhookLogs returns the seller's verified webhook rows.
	ListWebhookLogs(ctx context.Context, sellerID uint, filter repository.WebhookLogFilter) ([]entity.FulfillmentWebhookLog, int64, error)
}

// TrackingServiceImpl implements TrackingService.
type TrackingServiceImpl struct {
	shipmentRepo repository.ShipmentRepository
	eventRepo    repository.ShipmentEventRepository
	ndrRepo      repository.NDRRepository
	webhookRepo  repository.WebhookLogRepository
	orderHooks   FulfillmentOrderHooks
	couriers     *fulfillmentfactory.CourierPartnerFactory
	applier      *NormalizedApplier
	flight       *cachekit.Flight
}

// NewTrackingService builds the tracking service.
func NewTrackingService(
	shipmentRepo repository.ShipmentRepository,
	eventRepo repository.ShipmentEventRepository,
	ndrRepo repository.NDRRepository,
	webhookRepo repository.WebhookLogRepository,
	orderHooks FulfillmentOrderHooks,
	couriers *fulfillmentfactory.CourierPartnerFactory,
	applier *NormalizedApplier,
) TrackingService {
	return &TrackingServiceImpl{
		shipmentRepo: shipmentRepo,
		eventRepo:    eventRepo,
		ndrRepo:      ndrRepo,
		webhookRepo:  webhookRepo,
		orderHooks:   orderHooks,
		couriers:     couriers,
		applier:      applier,
		flight:       &cachekit.Flight{},
	}
}

// GetCustomerTrack returns the buyer's boxes: ownership-checked, sanitized
// (no seller id, no book-attempt internals, no failure codes, no NDR
// notes), events included. PG reads only.
func (s *TrackingServiceImpl) GetCustomerTrack(
	ctx context.Context,
	userID uint,
	orderID uint,
) ([]model.ShipmentResponse, error) {
	view, err := s.orderHooks.GetOrderForFulfillment(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if view.UserID != userID {
		return nil, fulfillmenterrors.ErrorFulfillmentNotFound
	}
	shipments, err := s.shipmentRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	currency := view.CurrencyCode
	if currency == "" {
		currency = "INR"
	}
	responses := make([]model.ShipmentResponse, 0, len(shipments))
	for i := range shipments {
		shipment := &shipments[i]
		mapped := model.ToShipmentResponse(shipment, shipment.Items, currency)
		mapped.SellerID = 0
		mapped.BookRequestedAt = nil
		mapped.BookAttempts = 0
		mapped.NDR = nil
		events, err := s.eventRepo.FindByShipmentID(ctx, shipment.ID)
		if err != nil {
			return nil, err
		}
		mapped.Events = make([]model.ShipmentEventResponse, 0, len(events))
		for _, event := range events {
			entry := model.ToShipmentEventResponse(event)
			entry.FailureCode = nil
			entry.FailureMessage = nil
			mapped.Events = append(mapped.Events, entry)
		}
		responses = append(responses, mapped)
	}
	return responses, nil
}

// RefreshTrack pulls live state and applies it. Concurrent refreshes of
// one box collapse onto a single provider call (singleflight); the apply
// stays idempotent regardless. Returns the full detail shape.
func (s *TrackingServiceImpl) RefreshTrack(
	ctx context.Context,
	sellerID uint,
	shipmentID uint,
) (model.ShipmentResponse, error) {
	shipment, err := s.shipmentRepo.FindByID(ctx, shipmentID)
	if err != nil {
		return model.ShipmentResponse{}, err
	}
	if shipment.SellerID != sellerID {
		return model.ShipmentResponse{}, fulfillmenterrors.ErrorFulfillmentNotFound
	}
	if shipment.ProviderConfigID == nil || shipment.AWB == nil {
		return model.ShipmentResponse{}, fulfillmenterrors.ErrorFulfillmentInvalidState.WithMessagef(
			"box has nothing to refresh yet")
	}
	if _, _, err := s.flightSingle(ctx, shipment); err != nil {
		return model.ShipmentResponse{}, err
	}
	return s.fullDetail(ctx, shipment.ID)
}

// flightSingle runs one provider fetch per box under singleflight.
func (s *TrackingServiceImpl) flightSingle(
	ctx context.Context,
	shipment *entity.FulfillmentShipment,
) (any, bool, error) {
	return s.flight.Do(refreshFlightKey(shipment.ID), func() (any, error) {
		return s.fetchAndApply(ctx, shipment)
	})
}

// fullDetail reloads a box with items, events, and the open NDR round in
// API shape (the refresh/post-apply read path).
func (s *TrackingServiceImpl) fullDetail(
	ctx context.Context,
	shipmentID uint,
) (model.ShipmentResponse, error) {
	shipment, err := s.shipmentRepo.FindByID(ctx, shipmentID)
	if err != nil {
		return model.ShipmentResponse{}, err
	}
	view, err := s.orderHooks.GetOrderForFulfillment(ctx, shipment.OrderID)
	currency := "INR"
	if err == nil && view.CurrencyCode != "" {
		currency = view.CurrencyCode
	}
	mapped := model.ToShipmentResponse(shipment, shipment.Items, currency)
	events, err := s.eventRepo.FindByShipmentID(ctx, shipment.ID)
	if err != nil {
		return model.ShipmentResponse{}, err
	}
	mapped.Events = make([]model.ShipmentEventResponse, 0, len(events))
	for _, event := range events {
		mapped.Events = append(mapped.Events, model.ToShipmentEventResponse(event))
	}
	ndr, err := s.ndrRepo.FindOpenByShipment(ctx, shipment.ID)
	if err != nil {
		return model.ShipmentResponse{}, err
	}
	mapped.NDR = model.ToNDRRoundResponse(ndr)
	return mapped, nil
}

// fetchAndApply is the singleflight body: one provider call + shared apply.
func (s *TrackingServiceImpl) fetchAndApply(
	ctx context.Context,
	shipment *entity.FulfillmentShipment,
) (*entity.FulfillmentShipment, error) {
	resolved, err := s.couriers.ResolveConfig(ctx, *shipment.ProviderConfigID)
	if err != nil {
		return nil, err
	}
	event, err := resolved.Adapter.FetchTracking(ctx, *shipment.AWB, resolved.Creds)
	if err != nil {
		return nil, err
	}
	if _, err := s.applier.Apply(ctx, shipment, event, EventSourceSystem); err != nil {
		return nil, err
	}
	return shipment, nil
}

func refreshFlightKey(shipmentID uint) string {
	return fmt.Sprintf("fulfill:refresh:%d", shipmentID)
}

// ListWebhookLogs returns verified rows linked to the seller's shipments.
func (s *TrackingServiceImpl) ListWebhookLogs(
	ctx context.Context,
	sellerID uint,
	filter repository.WebhookLogFilter,
) ([]entity.FulfillmentWebhookLog, int64, error) {
	filter.SellerID = sellerID
	logs, err := s.webhookRepo.FindBySeller(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.webhookRepo.CountBySeller(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
